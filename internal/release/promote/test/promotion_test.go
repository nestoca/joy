package promote_test

import (
	"fmt"
	"testing"

	"github.com/nestoca/joy/internal/info"
	"github.com/nestoca/joy/internal/links"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/release/promote"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
)

const (
	devEnvIndex     = 0
	stagingEnvIndex = 1
	prodEnvIndex    = 2
	sourceEnvIndex  = stagingEnvIndex
	targetEnvIndex  = prodEnvIndex
	sourceEnvName   = "staging"
	targetEnvName   = "prod"
)

type setupArgs struct {
	t              *testing.T
	opts           *promote.Opts
	gitProvider    *promote.MockGitProvider
	prProvider     *pr.MockPullRequestProvider
	promptProvider *promote.MockPromptProvider
	yamlWriter     *yml.WriterMock
	infoProvider   *info.MockProvider
	linksProvider  *links.MockProvider
}

func setupDefaultMockInfoProvider(provider *info.MockProvider) {
	provider.EXPECT().GetReleaseGitTag(gomock.Any()).Return("v1.0.0", nil).AnyTimes()
	provider.EXPECT().GetProjectRepository(gomock.Any()).Return("owner/project").AnyTimes()
	provider.EXPECT().GetProjectSourceDir(gomock.Any()).Return("/dummy/projects/project", nil).AnyTimes()
	provider.EXPECT().GetCommitsMetadata(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	provider.EXPECT().GetCommitsGitHubAuthors(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	provider.EXPECT().GetCodeOwners(gomock.Any()).Return([]string{"john-doe"}, nil).AnyTimes()
}

func setupDefaultMockLinksProvider(provider *links.MockProvider) {
	provider.EXPECT().GetReleaseLinks(gomock.Any()).Return(nil, nil).AnyTimes()
}

func TestPromotion(t *testing.T) {
	simpleCommitTemplate := "Commit: Promote {{ len .Releases }} releases ({{ .SourceEnvironment.Name }} -> {{ .TargetEnvironment.Name }})"
	simplePullRequestTemplate := "PR: Promote {{ len .Releases }} releases ({{ .SourceEnvironment.Name }} -> {{ .TargetEnvironment.Name }})"

	cases := []struct {
		name                    string
		opts                    promote.Opts
		setup                   func(args setupArgs)
		commitTemplate          string
		pullRequestTemplate     string
		pullRequestLinkTemplate string
		expectedErrorMessage    string
		expectedPromoted        bool
	}{
		{
			name: "Environment dev is not promotable to staging",
			opts: newOpts(),
			setup: func(args setupArgs) {
				opts := args.opts
				opts.SourceEnv = opts.Catalog.Environments[devEnvIndex]
				opts.TargetEnv = opts.Catalog.Environments[stagingEnvIndex]
			},
			expectedErrorMessage: "environment dev is not promotable to staging",
			expectedPromoted:     false,
		},
		{
			name: "Identical releases considered already in sync",
			opts: newOpts(),
			setup: func(args setupArgs) {
				opts := args.opts
				args.promptProvider.EXPECT().PrintNoPromotableReleasesFound(opts.ReleasesFiltered, opts.SourceEnv, opts.TargetEnv)
			},
			expectedPromoted: false,
		},
		{
			name: "Equivalent releases with different locked values are considered already in sync",
			opts: newOpts(),
			setup: func(args setupArgs) {
				opts := args.opts
				opts.Catalog.Releases.Items[0].Releases[sourceEnvIndex] = newRelease("release1", `spec:
  values:
    key: !lock value1`, sourceEnvName)
				opts.Catalog.Releases.Items[0].Releases[targetEnvIndex] = newRelease("release1", `spec:
  values:
    key: !lock value2`, targetEnvName)
				args.promptProvider.EXPECT().PrintNoPromotableReleasesFound(opts.ReleasesFiltered, opts.SourceEnv, opts.TargetEnv)
			},
			expectedPromoted: false,
		},
		{
			name: "Promote release1 from staging to prod",
			opts: newOpts(),
			setup: func(args setupArgs) {
				opts := args.opts
				crossRel0 := opts.Catalog.Releases.Items[0]
				targetEnv := opts.Catalog.Environments[targetEnvIndex]
				sourceRelease := newRelease("release1", `spec:
  values:
    key: !lock value1
    env:
      ENV_VAR: value1`, sourceEnvName)
				targetRelease := newRelease("release1", `spec:
  values:
    key: !lock value2
    env:
      ENV_VAR: value2`, targetEnvName)
				expectedPromotedFile := newYamlFile("release1", `spec:
  values:
    key: !lock value2
    env:
      ENV_VAR: value1
`, targetEnvName)

				crossRel0.Releases[sourceEnvIndex] = sourceRelease
				crossRel0.Releases[targetEnvIndex] = targetRelease

				args.promptProvider.EXPECT().SelectReleases(gomock.Any()).DoAndReturn(func(list *cross.ReleaseList) (*cross.ReleaseList, error) { return list, nil })
				args.promptProvider.EXPECT().PrintStartPreview()
				args.promptProvider.EXPECT().PrintReleasePreview(targetEnv.Name, crossRel0.Name, targetRelease.File, expectedPromotedFile)
				args.promptProvider.EXPECT().PrintEndPreview()
				args.promptProvider.EXPECT().SelectCreatingPromotionPullRequest().Return(promote.Ready, nil)
				args.promptProvider.EXPECT().PrintUpdatingTargetRelease(targetEnv.Name, crossRel0.Name, gomock.Any(), false)

				args.yamlWriter.WriteFileFunc = func(file *yml.File) error {
					return nil
				}
				// args.yamlWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(actualPromotedFile *yml.File) error {
				// 	expectedYaml, err := expectedPromotedFile.ToYaml()
				// 	assert.NoError(t, err)
				// 	assert.Equal(t, expectedYaml, string(actualPromotedFile.Yaml))
				// 	return nil
				// })
				args.gitProvider.EXPECT().CreateAndPushBranchWithFiles(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				args.promptProvider.EXPECT().PrintBranchCreated(gomock.Any(), gomock.Any())
				args.prProvider.EXPECT().Create(gomock.Any()).Return("https://github.com/owner/repo/pull/123", nil)
				args.promptProvider.EXPECT().PrintPullRequestCreated(gomock.Any())
				args.gitProvider.EXPECT().CheckoutMasterBranch().Return(nil)
				args.promptProvider.EXPECT().PrintCompleted()

				setupDefaultMockInfoProvider(args.infoProvider)
				setupDefaultMockLinksProvider(args.linksProvider)
			},
			commitTemplate:      simpleCommitTemplate,
			pullRequestTemplate: simplePullRequestTemplate,
			expectedPromoted:    true,
		},
		{
			name: "Promote release1 from staging to missing release in prod",
			opts: newOpts(),
			setup: func(args setupArgs) {
				opts := args.opts
				crossRel0 := opts.Catalog.Releases.Items[0]
				sourceEnv := opts.Catalog.Environments[sourceEnvIndex]
				targetEnv := opts.Catalog.Environments[targetEnvIndex]
				sourceRelease := newRelease("release1", `spec:
  values:
    key: !lock TODO
    env:
      ENV_VAR: value1`, sourceEnvName)

				sourceRelease.File.Path = fmt.Sprintf("%s/releases/testing/test.yaml", sourceEnv.Dir)

				crossRel0.Releases[sourceEnvIndex] = sourceRelease
				crossRel0.Releases[targetEnvIndex] = nil
				expectedPromotedFile := newYamlFile("release1", `spec:
  values:
    key: !lock TODO
    env:
      ENV_VAR: value1
`, targetEnvName)

				expectedPromotedFile.Path = fmt.Sprintf("%s/releases/testing/test.yaml", targetEnv.Dir)

				args.promptProvider.EXPECT().SelectReleases(gomock.Any()).DoAndReturn(func(list *cross.ReleaseList) (*cross.ReleaseList, error) { return list, nil })
				args.promptProvider.EXPECT().PrintStartPreview()
				args.promptProvider.EXPECT().PrintReleasePreview(targetEnv.Name, crossRel0.Name, nil, expectedPromotedFile)
				args.promptProvider.EXPECT().PrintEndPreview()
				args.promptProvider.EXPECT().SelectCreatingPromotionPullRequest().Return(promote.Ready, nil)
				args.promptProvider.EXPECT().PrintUpdatingTargetRelease(targetEnv.Name, crossRel0.Name, gomock.Any(), true)

				args.yamlWriter.WriteFileFunc = func(file *yml.File) error {
					return nil
				}

				// args.yamlWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(actualPromotedFile *yml.File) error {
				// 	expectedYaml, err := expectedPromotedFile.ToYaml()
				// 	assert.NoError(t, err)
				// 	assert.Equal(t, expectedYaml, string(actualPromotedFile.Yaml))
				// 	return nil
				// })

				args.gitProvider.EXPECT().CreateAndPushBranchWithFiles(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				args.promptProvider.EXPECT().PrintBranchCreated(gomock.Any(), gomock.Any())
				args.prProvider.EXPECT().Create(gomock.Any()).Return("https://github.com/owner/repo/pull/123", nil)
				args.promptProvider.EXPECT().PrintPullRequestCreated(gomock.Any())
				args.gitProvider.EXPECT().CheckoutMasterBranch().Return(nil)
				args.promptProvider.EXPECT().PrintCompleted()

				setupDefaultMockInfoProvider(args.infoProvider)
				setupDefaultMockLinksProvider(args.linksProvider)
			},
			commitTemplate:      simpleCommitTemplate,
			pullRequestTemplate: simplePullRequestTemplate,
			expectedPromoted:    true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Create mocks
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			gitProvider := promote.NewMockGitProvider(ctrl)
			prProvider := pr.NewMockPullRequestProvider(ctrl)
			promptProvider := promote.NewMockPromptProvider(ctrl)
			yamlWriter := new(yml.WriterMock)
			infoProvider := info.NewMockProvider(ctrl)
			linksProvider := links.NewMockProvider(ctrl)

			// Setup case-specific data and expectations
			c.setup(setupArgs{
				t:              t,
				opts:           &c.opts,
				gitProvider:    gitProvider,
				prProvider:     prProvider,
				promptProvider: promptProvider,
				yamlWriter:     yamlWriter,
				infoProvider:   infoProvider,
				linksProvider:  linksProvider,
			})

			// Perform test
			promotion := promote.Promotion{
				PromptProvider:      promptProvider,
				GitProvider:         gitProvider,
				PullRequestProvider: prProvider,
				YamlWriter:          yamlWriter,
				CommitTemplate:      simpleCommitTemplate,
				PullRequestTemplate: simplePullRequestTemplate,
				InfoProvider:        infoProvider,
				LinksProvider:       linksProvider,
			}
			prURL, err := promotion.Promote(c.opts)

			// Check expected results
			if c.expectedErrorMessage != "" {
				assert.EqualError(t, err, c.expectedErrorMessage)
			} else {
				assert.NoError(t, err)
			}
			if c.expectedPromoted {
				assert.NotEmpty(t, prURL)
			} else {
				assert.Empty(t, prURL)
			}
		})
	}
}

func newEnvironment(name string, promotionSourceEnvs ...string) *v1alpha1.Environment {
	return &v1alpha1.Environment{
		EnvironmentMetadata: v1alpha1.EnvironmentMetadata{
			Name: name,
		},
		Spec: v1alpha1.EnvironmentSpec{
			Promotion: v1alpha1.Promotion{
				FromEnvironments: promotionSourceEnvs,
			},
		},
		Dir: "/dummy/environments/" + name,
	}
}

func newEnvironments() []*v1alpha1.Environment {
	return []*v1alpha1.Environment{
		newEnvironment("dev"),
		newEnvironment("staging"),
		newEnvironment("prod", "staging"),
	}
}

var dummyProject = &v1alpha1.Project{
	ProjectMetadata: v1alpha1.ProjectMetadata{
		Name: "project1",
	},
	Spec: v1alpha1.ProjectSpec{
		Repository: "owner/project1",
	},
}

func newRelease(name, specYaml, envName string) *v1alpha1.Release {
	return &v1alpha1.Release{
		ReleaseMetadata: v1alpha1.ReleaseMetadata{
			Name: name,
		},
		File:    newYamlFile(name, specYaml, envName),
		Project: dummyProject,
	}
}

func newYamlFile(name, specYaml, envName string) *yml.File {
	if specYaml == "" {
		specYaml = `spec:
  values:
    key: value`
	}
	yaml := fmt.Sprintf(`apiVersion: joy.nesto.ca/v1alpha1
kind: Release
metadata:
  name: %s
%s`,
		name, specYaml)
	file, err := yml.NewFile("/dummy/environments/"+envName+"/releases/release.yaml", []byte(yaml))
	if err != nil {
		panic(err)
	}
	return file
}

func newCrossRelease(name string) *cross.Release {
	return &cross.Release{
		Name: name,
		Releases: []*v1alpha1.Release{
			newRelease(name, "", "dev"),
			newRelease(name, "", "staging"),
			newRelease(name, "", "prod"),
		},
	}
}

func newCatalog() *catalog.Catalog {
	envs := newEnvironments()
	return &catalog.Catalog{
		Environments: envs,
		Releases: &cross.ReleaseList{
			Environments: envs,
			Items: []*cross.Release{
				newCrossRelease("release1"),
				newCrossRelease("release2"),
			},
		},
	}
}

func newOpts() promote.Opts {
	cat := newCatalog()
	sourceEnv := cat.Environments[stagingEnvIndex]
	targetEnv := cat.Environments[prodEnvIndex]
	return promote.Opts{
		Catalog:              cat,
		SourceEnv:            sourceEnv,
		TargetEnv:            targetEnv,
		ReleasesFiltered:     false,
		SelectedEnvironments: nil,
	}
}
