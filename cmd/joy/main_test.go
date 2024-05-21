package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	cp "github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/dependencies"
	"github.com/nestoca/joy/internal/testutils"
	"github.com/nestoca/joy/pkg/catalog"
)

type TestRunParams struct {
	args                  []string
	version               string
	requiredDependencies  []*dependencies.Dependency
	optionalDependencies  []*dependencies.Dependency
	env                   map[string]string
	runFunc               func(t *testing.T, cmd *cobra.Command, args []string)
	omitDirFlags          bool
	commandUpdatesCatalog bool
}

func executeRun(t *testing.T, params TestRunParams, catalogDir string) (string, error) {
	args := []string{
		"test",
		"--skip-dev-check",
	}
	if !params.omitDirFlags {
		args = append(args,
			"--catalog-dir", catalogDir,
			"--config-dir", catalogDir,
		)
	}
	args = append(args, params.args...)

	var buffer bytes.Buffer
	io := internal.IO{
		Out: &buffer,
	}

	dependencies.AllRequired = params.requiredDependencies
	dependencies.AllOptional = params.optionalDependencies

	for key, value := range params.env {
		t.Setenv(key, value)
	}

	preRunConfigs := make(PreRunConfigs)

	err := run(RunParams{
		version: params.version,
		args:    args,
		io:      io,
		customizeRootCmdFunc: func(rootCmd *cobra.Command) error {
			testCmd := &cobra.Command{
				Use: "test",
				RunE: func(cmd *cobra.Command, args []string) error {
					params.runFunc(t, cmd, args)
					return nil
				},
			}
			if params.commandUpdatesCatalog {
				preRunConfigs.PullCatalog(testCmd)
			}
			rootCmd.AddCommand(testCmd)
			return nil
		},
		preRunConfigs: preRunConfigs,
	})
	return stripansi.Strip(buffer.String()), err
}

type TestCase struct {
	name          string
	params        *TestRunParams
	expectedOut   string
	expectedError string
	setupFunc     func(t *testing.T, tc *TestCase, catalogDir string)
}

func TestMainRun(t *testing.T) {
	testCases := []TestCase{
		{
			name: "missing required dependency",
			params: &TestRunParams{
				requiredDependencies: []*dependencies.Dependency{{Command: "dummy", Url: "https://dummy.com", IsRequired: true}},
			},
			expectedError: "😅 Oops! Joy requires those dependencies to operate. Please install them and try again! 🙏",
		},
		{
			name: "fulfilled required dependency",
			params: &TestRunParams{
				requiredDependencies: []*dependencies.Dependency{{Command: "ls", Url: "https://ls.com", IsRequired: true}},
			},
		},
		{
			name: "insufficient min version",
			params: &TestRunParams{
				version: "v0.1.1",
			},
			expectedError: "Current version \"v0.1.1\" is less than required minimum version \"v0.1.2\"\n\nPlease update joy! >> brew update && brew upgrade joy",
		},
		{
			name: "insufficient min version check skipped via flag",
			params: &TestRunParams{
				version: "v0.1.1",
				args:    []string{"--skip-version-check"},
			},
		},
		{
			name: "insufficient min version check skipped via env var",
			params: &TestRunParams{
				version: "v0.1.1",
				env:     map[string]string{"JOY_DEV_SKIP_VERSION_CHECK": "1"},
			},
		},
		{
			name: "sufficient min version",
			params: &TestRunParams{
				version: "v0.1.2",
			},
		},
		{
			name: "load catalog from default user home location",
			params: &TestRunParams{
				omitDirFlags: true,
			},
			setupFunc: func(t *testing.T, tc *TestCase, catalogDir string) {
				homeDir := filepath.Dir(catalogDir)
				tc.params.env = map[string]string{"HOME": homeDir}
			},
		},
		{
			name: "load catalog from location specified in default user config file",
			params: &TestRunParams{
				omitDirFlags: true,
			},
			setupFunc: func(t *testing.T, tc *TestCase, catalogDir string) {
				homeDir := filepath.Dir(catalogDir)
				customCatalogDir := filepath.Join(homeDir, "custom-catalog-dir")
				require.NoError(t, os.Rename(catalogDir, customCatalogDir))
				require.NoError(t, os.WriteFile(filepath.Join(homeDir, ".joyrc"), []byte("catalogDir: "+customCatalogDir), 0o644))
				tc.params.env = map[string]string{"HOME": homeDir}
				tc.params.runFunc = func(t *testing.T, cmd *cobra.Command, args []string) {
					require.Equal(t, customCatalogDir, config.FromContext(cmd.Context()).CatalogDir)
					requireValidCatalog(t, catalog.FromContext(cmd.Context()))
				}
			},
		},
		{
			name: "command that normally updates catalog should do it in absence of --skip-catalog-update flag",
			params: &TestRunParams{
				commandUpdatesCatalog: true,
			},
			setupFunc: setupFuncToRequireCatalogUpdate,
		},
		{
			name: "command that normally updates catalog should not do it in presence of --skip-catalog-update flag",
			params: &TestRunParams{
				args:                  []string{"--skip-catalog-update"},
				commandUpdatesCatalog: true,
			},
			setupFunc: setupFuncToRequireNoCatalogUpdate,
		},
		{
			name: "command that normally updates catalog should fail in the presence of uncommitted changes",
			params: &TestRunParams{
				commandUpdatesCatalog: true,
			},
			setupFunc: func(t *testing.T, tc *TestCase, catalogDir string) {
				_, err := os.Create(filepath.Join(catalogDir, "dummy"))
				require.NoError(t, err)
			},
			expectedError: "uncommitted catalog changes detected",
		},
		{
			name: "command that normally does not update catalog should not do it in absence of --skip-catalog-update flag",
			params: &TestRunParams{
				commandUpdatesCatalog: false,
			},
			setupFunc: setupFuncToRequireNoCatalogUpdate,
		},
	}

	originalCatalogDir := testutils.CloneToTempDir(t, "joy-release-promote-test")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempHomeDir := t.TempDir()
			catalogDir := filepath.Join(tempHomeDir, ".joy")
			require.NoError(t, cp.Copy(originalCatalogDir, catalogDir))

			tc.params.runFunc = func(t *testing.T, cmd *cobra.Command, args []string) {
				require.Equal(t, catalogDir, config.FromContext(cmd.Context()).CatalogDir)
				requireValidCatalog(t, catalog.FromContext(cmd.Context()))
			}

			if tc.setupFunc != nil {
				tc.setupFunc(t, &tc, catalogDir)
			}

			out, err := executeRun(t, *tc.params, catalogDir)

			if tc.expectedOut != "" {
				require.Contains(t, out, tc.expectedOut)
			}

			if tc.expectedError != "" {
				require.NotNil(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func cloneToTempDir(t *testing.T, repoName string) string {
	dir := t.TempDir()
	repoURL := func() string {
		if gitToken := os.Getenv("GH_TOKEN"); gitToken != "" {
			return fmt.Sprintf("https://%s@github.com/nestoca/%s.git", gitToken, repoName)
		}
		return fmt.Sprintf("git@github.com:nestoca/%s.git", repoName)
	}()
	cmd := exec.Command("git", "clone", repoURL, dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	require.NoError(t, cmd.Run())
	return dir
}

func setupFuncToRequireCatalogUpdate(t *testing.T, tc *TestCase, dir string) {
	masterHash := getMasterHeadHash(t, dir)
	checkOutRef(t, dir, "master~1")
	defaultRunFunc := tc.params.runFunc
	tc.params.runFunc = func(t *testing.T, cmd *cobra.Command, args []string) {
		defaultRunFunc(t, cmd, args)
		currentHash := getCurrentHash(t, dir)
		require.Equal(t, masterHash, currentHash, "not on master head")
	}
}

func setupFuncToRequireNoCatalogUpdate(t *testing.T, tc *TestCase, dir string) {
	checkOutRef(t, dir, "master~1")
	hashBefore := getCurrentHash(t, dir)
	defaultRunFunc := tc.params.runFunc
	tc.params.runFunc = func(t *testing.T, cmd *cobra.Command, args []string) {
		defaultRunFunc(t, cmd, args)
		hashAfter := getCurrentHash(t, dir)
		require.Equal(t, hashBefore, hashAfter, "unexpected catalog update")
	}
}

func getMasterHeadHash(t *testing.T, dir string) string {
	r, err := git.PlainOpen(dir)
	require.NoError(t, err)

	masterRef, err := r.Reference("refs/heads/master", true)
	require.NoError(t, err)

	return masterRef.Hash().String()
}

func getCurrentHash(t *testing.T, dir string) string {
	r, err := git.PlainOpen(dir)
	require.NoError(t, err)

	headRef, err := r.Head()
	require.NoError(t, err)

	return headRef.Hash().String()
}

func checkOutRef(t *testing.T, dir, ref string) {
	r, err := git.PlainOpen(dir)
	require.NoError(t, err)

	w, err := r.Worktree()
	require.NoError(t, err)

	err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(ref),
	})
	require.NoError(t, err)
}

func requireValidCatalog(t *testing.T, cat *catalog.Catalog) {
	require.NotNil(t, cat)
	require.Greater(t, len(cat.Environments), 0)
}
