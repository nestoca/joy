package build_test

import (
	"fmt"
	"github.com/nestoca/joy-cli/internal/build"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

const releaseTemplate = `# Some random comment
apiVersion: joy.nesto.ca/v1alpha1
kind: Release
# another random comment
metadata:
  annotations: {}
  name: %s
spec:
  chart:
    name: echo
    # a nested comment
    repoUrl: https://repo.echo-chart.example
    version: 1.2.3
  project: %s
  version: %s # This is an line comment
  versionKey: image.tag
`

func TestPromote(t *testing.T) {
	testArgs := build.PromoteArgs{
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.0.0-updated",
		CatalogDir:  t.TempDir(),
	}

	testReleaseFile, err := setupPromoteTest(setupPromoteTestArgs{
		catalogDir: testArgs.CatalogDir,
		env:        testArgs.Environment,
		project:    testArgs.Project,
	})
	assert.Nil(t, err)

	err = build.Promote(testArgs)
	assert.Nil(t, err)

	result, err := os.ReadFile(testReleaseFile)
	assert.Nil(t, err)

	assert.Equal(t,
		fmt.Sprintf(releaseTemplate, "promote-build-release", testArgs.Project, testArgs.Version),
		string(result),
	)
}

func TestPromoteWhenNoReleasesFoundForProject(t *testing.T) {
	testArgs := build.PromoteArgs{
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.0.0-updated",
		CatalogDir:  t.TempDir(),
	}

	testReleaseFile, err := setupPromoteTest(setupPromoteTestArgs{
		catalogDir: testArgs.CatalogDir,
		env:        testArgs.Environment,
		project:    "other-project",
	})
	assert.Nil(t, err)

	err = build.Promote(testArgs)
	assert.NotNil(t, err)

	result, err := os.ReadFile(testReleaseFile)
	assert.Nil(t, err)

	// Should be unchanged
	assert.Equal(t,
		fmt.Sprintf(releaseTemplate, "promote-build-release", "other-project", "0.0.1"),
		string(result),
	)
}

func TestPromoteWhenCatalogDirNotExists(t *testing.T) {
	testArgs := build.PromoteArgs{
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.0.0-updated",
		CatalogDir:  "/foo/bar",
	}

	err := build.Promote(testArgs)
	assert.NotNil(t, err)
}

func TestPromoteWhenReleaseYamlProjectPathDoesNotExists(t *testing.T) {
	testArgs := build.PromoteArgs{
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.0.0-updated",
		CatalogDir:  t.TempDir(),
	}

	fileContents := `some:
  other:
    yaml: foo
  document: bar
`

	testReleaseFile, err := setupPromoteTest(setupPromoteTestArgs{
		catalogDir:   testArgs.CatalogDir,
		env:          testArgs.Environment,
		fileContents: fileContents,
	})
	assert.Nil(t, err)

	err = build.Promote(testArgs)
	assert.NotNil(t, err)
	assert.EqualError(t,
		err,
		"walking catalog directory: reading release's project: node not found for path '.spec.project': key 'spec' does not exist",
	)

	result, err := os.ReadFile(testReleaseFile)
	assert.Nil(t, err)

	// File should be unchanged
	assert.Equal(t,
		fileContents,
		string(result),
	)
}

func TestPromoteWhenReleaseYamlVersionPathDoesNotExists(t *testing.T) {
	testArgs := build.PromoteArgs{
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.0.0-updated",
		CatalogDir:  t.TempDir(),
	}

	fileContents := `spec:
  project: promote-build
  other:
    foo: foobar
  document: bar
`

	testReleaseFile, err := setupPromoteTest(setupPromoteTestArgs{
		catalogDir:   testArgs.CatalogDir,
		env:          testArgs.Environment,
		fileContents: fileContents,
	})
	assert.Nil(t, err)

	err = build.Promote(testArgs)
	assert.NotNil(t, err)
	assert.EqualError(t,
		err,
		"updating release version: node not found for path '.spec.version': key 'version' does not exist",
	)

	result, err := os.ReadFile(testReleaseFile)
	assert.Nil(t, err)

	// File should be unchanged
	assert.Equal(t,
		fileContents,
		string(result),
	)
}

type setupPromoteTestArgs struct {
	catalogDir   string
	env          string
	project      string
	fileContents string
}

func setupPromoteTest(args setupPromoteTestArgs) (string, error) {
	testReleaseDir := filepath.Join(
		args.catalogDir,
		"environments",
		args.env,
		"releases",
	)
	err := os.MkdirAll(testReleaseDir, 0755)
	if err != nil {
		return "", err
	}

	testReleaseFile := filepath.Join(
		testReleaseDir,
		"promote-build.release.yaml",
	)

	if args.fileContents == "" {
		args.fileContents = fmt.Sprintf(releaseTemplate, "promote-build-release", args.project, "0.0.1")
	}

	err = os.WriteFile(
		testReleaseFile,
		[]byte(args.fileContents),
		0644,
	)
	if err != nil {
		return "", err
	}

	return testReleaseFile, nil
}
