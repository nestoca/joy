package build

import (
	"fmt"
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
  version: %s # This is a line comment
  versionKey: image.tag
`

const environmentTemplate = `apiVersion: joy.nesto.ca/v1alpha1
kind: Environment
metadata:
  name: %s
`

func TestPromote(t *testing.T) {
	opts := Opts{
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.0.0-updated",
	}

	testReleaseFile, err := setup(setupOpts{
		env:     opts.Environment,
		project: opts.Project,
	})
	assert.Nil(t, err)

	err = Promote(opts)
	assert.Nil(t, err)

	result, err := os.ReadFile(testReleaseFile)
	assert.Nil(t, err)

	assert.Equal(t,
		fmt.Sprintf(releaseTemplate, "promote-build-release", opts.Project, opts.Version),
		string(result),
	)
}

func TestPromoteWhenNoReleasesFoundForProject(t *testing.T) {
	opts := Opts{
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.0.0-updated",
	}

	testReleaseFile, err := setup(setupOpts{
		env:     opts.Environment,
		project: "other-project",
	})
	assert.NoError(t, err)

	err = Promote(opts)
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
	opts := Opts{
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.0.0-updated",
	}

	err := Promote(opts)
	assert.NotNil(t, err)
}

func TestPromoteWhenReleaseYamlProjectPathDoesNotExists(t *testing.T) {
	opts := Opts{
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.0.0-updated",
	}

	fileContents := `some:
  other:
    yaml: foo
  document: bar
`

	testReleaseFile, err := setup(setupOpts{
		env:          opts.Environment,
		fileContents: fileContents,
	})
	assert.Nil(t, err)

	err = Promote(opts)
	assert.NotNil(t, err)
	assert.EqualError(t,
		err,
		"no releases found for project promote-build",
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
	opts := Opts{
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.0.0-updated",
	}

	fileContents := `spec:
  project: promote-build
  other:
    foo: foobar
  document: bar
`

	testReleaseFile, err := setup(setupOpts{
		env:          opts.Environment,
		fileContents: fileContents,
	})
	assert.Nil(t, err)

	err = Promote(opts)
	assert.NotNil(t, err)
	assert.EqualError(t,
		err,
		"no releases found for project promote-build",
	)

	result, err := os.ReadFile(testReleaseFile)
	assert.Nil(t, err)

	// File should be unchanged
	assert.Equal(t,
		fileContents,
		string(result),
	)
}

type setupOpts struct {
	env          string
	project      string
	fileContents string
}

func setup(opts setupOpts) (string, error) {
	// Create temp catalog dir
	tempDir, err := os.MkdirTemp("", "promote-test-")
	if err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}
	os.Chdir(tempDir)

	// Create environment file
	environmentFile := filepath.Join(tempDir, "env.yaml")
	environmentContent := fmt.Sprintf(environmentTemplate, opts.env)
	err = os.WriteFile(environmentFile, []byte(environmentContent), 0644)
	if err != nil {
		return "", fmt.Errorf("writing environment file: %w", err)
	}

	// Create release file
	releaseFile := filepath.Join(tempDir, "promote-build.yaml")
	if opts.fileContents == "" {
		opts.fileContents = fmt.Sprintf(releaseTemplate, "promote-build-release", opts.project, "0.0.1")
	}
	err = os.WriteFile(releaseFile, []byte(opts.fileContents), 0644)
	if err != nil {
		return "", err
	}

	return releaseFile, nil
}
