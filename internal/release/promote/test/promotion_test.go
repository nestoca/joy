package promote_test

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/git/pr"
	"github.com/nestoca/joy/internal/info"
	"github.com/nestoca/joy/internal/links"
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
	gitProvider    *promote.GitProviderMock
	prProvider     *pr.PullRequestProviderMock
	promptProvider *promote.PromptProviderMock
	yamlWriter     *yml.WriterMock
	infoProvider   *info.ProviderMock
	linksProvider  *links.ProviderMock
}

func setupDefaultMockInfoProvider(provider *info.ProviderMock) {
	*provider = info.ProviderMock{
		GetCommitsGitHubAuthorsFunc: func(project *v1alpha1.Project, fromTag string, toTag string) (map[string]string, error) {
			return nil, nil
		},
		GetCommitsMetadataFunc: func(projectDir string, fromTag string, toTag string) ([]*info.CommitMetadata, error) {
			return nil, nil
		},
		GetProjectRepositoryFunc: func(project *v1alpha1.Project) string {
			return "owner/project"
		},
		GetProjectSourceDirFunc: func(project *v1alpha1.Project) (string, error) {
			return "/dummy/projects/project", nil
		},
		GetReleaseGitTagFunc: func(release *v1alpha1.Release) (string, error) {
			return "v1.0.0", nil
		},
	}
}

func TestPromotion(t *testing.T) {
	simpleCommitTemplate := "Commit: Promote {{ len .Releases }} releases ({{ .SourceEnvironment.Name }} -> {{ .TargetEnvironment.Name }})"
	simplePullRequestTemplate := "PR: Promote {{ len .Releases }} releases ({{ .SourceEnvironment.Name }} -> {{ .TargetEnvironment.Name }})"

	cases := []struct {
		name                    string
		opts                    promote.Opts
		setup                   func(args setupArgs) func(t *testing.T)
		commitTemplate          string
		pullRequestTemplate     string
		pullRequestLinkTemplate string
		expectedErrorMessage    string
		expectedPromoted        bool
	}{
		{
			name: "Environment dev is not promotable to staging",
			opts: newOpts(),
			setup: func(args setupArgs) func(t *testing.T) {
				opts := args.opts
				opts.SourceEnv = opts.Catalog.Environments[devEnvIndex]
				opts.TargetEnv = opts.Catalog.Environments[stagingEnvIndex]

				return func(t *testing.T) {}
			},
			expectedErrorMessage: "environment dev is not promotable to staging",
			expectedPromoted:     false,
		},
		{
			name: "Identical releases considered already in sync",
			opts: newOpts(),
			setup: func(args setupArgs) func(t *testing.T) {
				return func(t *testing.T) {
					require.Len(t, args.promptProvider.PrintNoPromotableReleasesFoundCalls(), 1)
					call := args.promptProvider.PrintNoPromotableReleasesFoundCalls()[0]
					require.Equal(t, args.opts.ReleasesFiltered, call.ReleasesFiltered)
					require.Equal(t, args.opts.SourceEnv, call.SourceEnv)
					require.Equal(t, args.opts.TargetEnv, call.TargetEnv)
				}
			},
			expectedPromoted: false,
		},
		{
			name: "Equivalent releases with different locked values are considered already in sync",
			opts: newOpts(),
			setup: func(args setupArgs) func(t *testing.T) {
				opts := args.opts
				opts.Catalog.Releases.Items[0].Releases[sourceEnvIndex] = newRelease("release1", `spec:
  values:
    key: !lock value1`, sourceEnvName)
				opts.Catalog.Releases.Items[0].Releases[targetEnvIndex] = newRelease("release1", `spec:
  values:
    key: !lock value2`, targetEnvName)

				return func(t *testing.T) {
					require.Len(t, args.promptProvider.PrintNoPromotableReleasesFoundCalls(), 1)
					call := args.promptProvider.PrintNoPromotableReleasesFoundCalls()[0]
					require.Equal(t, args.opts.ReleasesFiltered, call.ReleasesFiltered)
					require.Equal(t, args.opts.SourceEnv, call.SourceEnv)
					require.Equal(t, args.opts.TargetEnv, call.TargetEnv)
				}
			},
			expectedPromoted: false,
		},
		{
			name: "Promote release1 from staging to prod",
			opts: newOpts(),
			setup: func(args setupArgs) func(t *testing.T) {
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

				args.promptProvider.SelectReleasesFunc = func(list cross.ReleaseList, maxColumnWidth int) (cross.ReleaseList, error) {
					return list, nil
				}
				args.promptProvider.SelectCreatingPromotionPullRequestFunc = func() (string, error) {
					return promote.Ready, nil
				}

				args.yamlWriter.WriteFileFunc = func(file *yml.File) error {
					return nil
				}

				args.prProvider.CreateFunc = func(createParams pr.CreateParams) (string, error) {
					return "https://github.com/owner/repo/pull/123", nil
				}

				setupDefaultMockInfoProvider(args.infoProvider)

				return func(t *testing.T) {
					require.Len(t, args.promptProvider.SelectReleasesCalls(), 1)
					require.Len(t, args.promptProvider.PrintStartPreviewCalls(), 1)
					require.Len(t, args.promptProvider.PrintReleasePreviewCalls(), 1)

					printReleasePreviewCall := args.promptProvider.PrintReleasePreviewCalls()[0]
					require.Equal(t, targetEnv.Name, printReleasePreviewCall.TargetEnvName)
					require.Equal(t, crossRel0.Name, printReleasePreviewCall.ReleaseName)
					require.Equal(t, targetRelease.File, printReleasePreviewCall.ExistingTargetFile)
					require.Equal(t, expectedPromotedFile, printReleasePreviewCall.PromotedFile)

					require.Len(t, args.promptProvider.PrintEndPreviewCalls(), 1)
					require.Len(t, args.promptProvider.SelectCreatingPromotionPullRequestCalls(), 1)

					require.Len(t, args.promptProvider.PrintUpdatingTargetReleaseCalls(), 1)
					printUpdatingCall := args.promptProvider.PrintUpdatingTargetReleaseCalls()[0]
					require.Equal(t, targetEnv.Name, printUpdatingCall.TargetEnvName)
					require.Equal(t, crossRel0.Name, printUpdatingCall.ReleaseName)
					require.Equal(t, false, printUpdatingCall.IsCreating)

					require.Len(t, args.promptProvider.PrintBranchCreatedCalls(), 1)
					require.Len(t, args.promptProvider.PrintPullRequestCreatedCalls(), 1)
					require.Len(t, args.promptProvider.PrintCompletedCalls(), 1)
				}
			},
			commitTemplate:      simpleCommitTemplate,
			pullRequestTemplate: simplePullRequestTemplate,
			expectedPromoted:    true,
		},
		{
			name: "Promote release1 from staging to missing release in prod",
			opts: newOpts(),
			setup: func(args setupArgs) func(t *testing.T) {
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

				args.promptProvider.SelectReleasesFunc = func(list cross.ReleaseList, maxColumnWidth int) (cross.ReleaseList, error) {
					return list, nil
				}
				args.promptProvider.SelectCreatingPromotionPullRequestFunc = func() (string, error) {
					return promote.Ready, nil
				}

				args.yamlWriter.WriteFileFunc = func(file *yml.File) error {
					return nil
				}

				args.prProvider.CreateFunc = func(createParams pr.CreateParams) (string, error) {
					return "https://github.com/owner/repo/pull/123", nil
				}

				setupDefaultMockInfoProvider(args.infoProvider)

				return func(t *testing.T) {
					require.Len(t, args.promptProvider.SelectReleasesCalls(), 1)
					require.Len(t, args.promptProvider.PrintStartPreviewCalls(), 1)
					require.Len(t, args.promptProvider.PrintReleasePreviewCalls(), 1)

					printReleasePreviewCall := args.promptProvider.PrintReleasePreviewCalls()[0]
					require.Equal(t, targetEnv.Name, printReleasePreviewCall.TargetEnvName)
					require.Equal(t, crossRel0.Name, printReleasePreviewCall.ReleaseName)
					require.Equal(t, (*yml.File)(nil), printReleasePreviewCall.ExistingTargetFile)
					require.Equal(t, expectedPromotedFile, printReleasePreviewCall.PromotedFile)

					require.Len(t, args.promptProvider.PrintEndPreviewCalls(), 1)
					require.Len(t, args.promptProvider.SelectCreatingPromotionPullRequestCalls(), 1)

					require.Len(t, args.promptProvider.PrintUpdatingTargetReleaseCalls(), 1)
					printUpdatingCall := args.promptProvider.PrintUpdatingTargetReleaseCalls()[0]
					require.Equal(t, targetEnv.Name, printUpdatingCall.TargetEnvName)
					require.Equal(t, crossRel0.Name, printUpdatingCall.ReleaseName)
					require.Equal(t, true, printUpdatingCall.IsCreating)

					require.Len(t, args.promptProvider.PrintBranchCreatedCalls(), 1)
					require.Len(t, args.promptProvider.PrintPullRequestCreatedCalls(), 1)
					require.Len(t, args.promptProvider.PrintCompletedCalls(), 1)
				}
			},
			commitTemplate:      simpleCommitTemplate,
			pullRequestTemplate: simplePullRequestTemplate,
			expectedPromoted:    true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gitProvider := new(promote.GitProviderMock)
			prProvider := new(pr.PullRequestProviderMock)
			promptProvider := new(promote.PromptProviderMock)
			yamlWriter := new(yml.WriterMock)
			infoProvider := new(info.ProviderMock)
			linksProvider := new(links.ProviderMock)

			// Setup case-specific data and expectations
			defer c.setup(setupArgs{
				t:              t,
				opts:           &c.opts,
				gitProvider:    gitProvider,
				prProvider:     prProvider,
				promptProvider: promptProvider,
				yamlWriter:     yamlWriter,
				infoProvider:   infoProvider,
				linksProvider:  linksProvider,
			})(t)

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
				Out:                 io.Discard,
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
		Releases: cross.ReleaseList{
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
