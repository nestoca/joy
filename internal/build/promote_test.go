package build

import (
	"fmt"
	"github.com/stretchr/testify/require"
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
		Version:     "1.1.2",
	}

	testReleaseFile, err := setup(setupOpts{
		env:     opts.Environment,
		project: opts.Project,
	})
	require.NoError(t, err)

	err = Promote(opts)
	require.NoError(t, err)

	result, err := os.ReadFile(testReleaseFile)
	require.NoError(t, err)

	require.Equal(t,
		fmt.Sprintf(releaseTemplate, "promote-build-release", opts.Project, opts.Version),
		string(result),
	)
}

func TestPromoteWithInvalidVersion(t *testing.T) {
	opts := Opts{
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.1.2-updated",
	}

	_, err := setup(setupOpts{
		env:     opts.Environment,
		project: opts.Project,
	})
	require.NoError(t, err)

	err = Promote(opts)
	require.EqualError(t, err, "cannot promote release with non-standard version to testing environment")
}

func TestPromoteWhenNoReleasesFoundForProject(t *testing.T) {
	opts := Opts{
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.1.2",
	}

	testReleaseFile, err := setup(setupOpts{
		env:     opts.Environment,
		project: "other-project",
	})
	require.NoError(t, err)

	err = Promote(opts)
	require.EqualError(t, err, "no releases found for project promote-build")

	result, err := os.ReadFile(testReleaseFile)
	require.NoError(t, err)

	// Should be unchanged
	require.Equal(t,
		fmt.Sprintf(releaseTemplate, "promote-build-release", "other-project", "0.0.1"),
		string(result),
	)
}

func TestPromoteWhenCatalogDirNotExists(t *testing.T) {
	opts := Opts{
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.1.2",
	}

	err := Promote(opts)
	require.NotNil(t, err)
}

func TestPromoteWhenReleaseYamlProjectPathDoesNotExists(t *testing.T) {
	opts := Opts{
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.1.2",
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
	require.NoError(t, err)

	err = Promote(opts)
	require.EqualError(t, err, "no releases found for project promote-build")

	result, err := os.ReadFile(testReleaseFile)
	require.NoError(t, err)

	// File should be unchanged
	require.Equal(t,
		fileContents,
		string(result),
	)
}

func TestPromoteWhenReleaseYamlVersionPathDoesNotExists(t *testing.T) {
	opts := Opts{
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.1.2",
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
	require.NoError(t, err)

	err = Promote(opts)
	require.EqualError(t, err, "no releases found for project promote-build")

	result, err := os.ReadFile(testReleaseFile)
	require.NoError(t, err)

	// File should be unchanged
	require.Equal(t,
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
	err = os.WriteFile(environmentFile, []byte(environmentContent), 0o644)
	if err != nil {
		return "", fmt.Errorf("writing environment file: %w", err)
	}

	// Create release file
	releaseFile := filepath.Join(tempDir, "promote-build.yaml")
	if opts.fileContents == "" {
		opts.fileContents = fmt.Sprintf(releaseTemplate, "promote-build-release", opts.project, "0.0.1")
	}
	err = os.WriteFile(releaseFile, []byte(opts.fileContents), 0o644)
	if err != nil {
		return "", err
	}

	return releaseFile, nil
}
