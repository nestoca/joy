package diagnostics

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
)

func TestCatalogDiagnostics(t *testing.T) {
	cases := []struct {
		Name       string
		CatalogRef string
		Opts       CatalogOpts
		Expected   Group
	}{
		{
			Name: "happy",
			Opts: CatalogOpts{
				Stat: func(string) (fs.FileInfo, error) { return nil, nil },
				Git: GitOpts{
					IsValid:               func(string) bool { return true },
					GetUncommittedChanges: func(string) ([]string, error) { return nil, nil },
					GetCurrentBranch:      func(string) (string, error) { return "master", nil },
					IsInSyncWithRemote:    func(string, string) (bool, error) { return true, nil },
					GetCurrentCommit:      func(string) (string, error) { return "origin/HEAD", nil },
					IsRemoteBranch:        func(string, string) (bool, error) { return true, nil },
				},
				CheckCatalog: func(s string) error { return nil },
				LoadCatalog: func(context.Context, string, []string) (*catalog.Catalog, error) {
					return &catalog.Catalog{
						Environments: []*v1alpha1.Environment{},
						Releases:     cross.ReleaseList{},
						Projects:     []*v1alpha1.Project{},
						Files:        []*yml.File{},
					}, nil
				},
			},
			Expected: Group{
				Title: "Catalog",
				SubGroups: Groups{
					{
						Title: "Git working copy", Messages: Messages{
							{Type: "info", Value: "Directory exists: catalogDir"},
							{Type: "success", Value: "Working copy is valid"},
							{Type: "success", Value: "Working copy has no uncommitted changes"},
							{Type: "success", Value: "Catalog branch master is checked out"},
							{Type: "success", Value: "Catalog branch is in sync with remote"},
							{Type: "info", Value: "Current commit: origin/HEAD"},
						},
					},
					{
						Title: "Loading catalog", Messages: Messages{
							{Type: "success", Value: "Catalog detected"},
							{Type: "success", Value: "Catalog loaded successfully"},
						},
					},
					{
						Title: "Resources", Messages: Messages{
							{Type: "info", Value: "Environments: 0"},
							{Type: "info", Value: "Projects: 0"},
							{Type: "info", Value: "Releases: 0"},
						},
					},
					{
						Title: "Cross-references", Messages: Messages{
							{Type: "success", Value: "All resource cross-references resolved successfully"},
						},
					},
				},
				topLevel: true,
			},
		},
		{
			Name: "catalog not exists",
			Opts: CatalogOpts{
				Stat:         func(s string) (fs.FileInfo, error) { return nil, os.ErrNotExist },
				CheckCatalog: func(s string) error { return errors.New("no joy catalog found at \"catalogDir\"") },
			},
			Expected: Group{
				Title: "Catalog",
				SubGroups: Groups{
					{
						Title: "Git working copy",
						Messages: Messages{
							{Type: "failed", Value: "Directory does not exist: catalogDir"},
						},
					},
					{
						Title: "Loading catalog",
						Messages: Messages{
							{Type: "failed", Value: "Catalog not detected: no joy catalog found at \"catalogDir\""},
						},
					},
				},
				topLevel: true,
			},
		},
		{
			Name: "git not valid repository",
			Opts: CatalogOpts{
				Stat:         func(s string) (fs.FileInfo, error) { return nil, nil },
				CheckCatalog: func(s string) error { return errors.New("no joy catalog found at \"catalogDir\"") },
				Git: GitOpts{
					IsValid: func(s string) bool { return false },
				},
			},
			Expected: Group{
				Title: "Catalog",
				SubGroups: Groups{
					{
						Title: "Git working copy", Messages: Messages{
							{Type: "info", Value: "Directory exists: catalogDir"},
							{Type: "failed", Value: "Working copy is invalid"},
						},
					},
					{
						Title: "Loading catalog", Messages: Messages{
							{Type: "failed", Value: "Catalog not detected: no joy catalog found at \"catalogDir\""},
						},
					},
				}, topLevel: true,
			},
		},
		{
			Name:       "detached catalog ref checked out",
			CatalogRef: "v1",
			Opts: CatalogOpts{
				Stat:         func(string) (fs.FileInfo, error) { return nil, nil },
				CheckCatalog: func(string) error { return errors.New("no joy catalog found at \"catalogDir\"") },
				Git: GitOpts{
					IsValid:               func(string) bool { return true },
					GetUncommittedChanges: func(string) ([]string, error) { return nil, nil },
					GetCurrentBranch:      func(string) (string, error) { return "HEAD", nil },
					IsInSyncWithRemote: func(string, string) (bool, error) {
						return false, errors.New("should not be called for detached refs")
					},
					GetCurrentCommit: func(string) (string, error) { return "abc123", nil },
					IsRemoteBranch:   func(string, string) (bool, error) { return false, nil },
					ResolveCommit:    func(string, string) (string, error) { return "abc123", nil },
				},
			},
			Expected: Group{
				Title: "Catalog",
				SubGroups: Groups{
					{
						Title: "Git working copy", Messages: Messages{
							{Type: "info", Value: "Directory exists: catalogDir"},
							{Type: "success", Value: "Working copy is valid"},
							{Type: "success", Value: "Working copy has no uncommitted changes"},
							{Type: "success", Value: "Catalog ref v1 is checked out"},
							{Type: "info", Value: "Current commit: abc123"},
						},
					},
					{
						Title: "Loading catalog", Messages: Messages{
							{Type: "failed", Value: "Catalog not detected: no joy catalog found at \"catalogDir\""},
						},
					},
				}, topLevel: true,
			},
		},
		{
			Name:       "detached catalog ref not checked out",
			CatalogRef: "v1",
			Opts: CatalogOpts{
				Stat:         func(string) (fs.FileInfo, error) { return nil, nil },
				CheckCatalog: func(string) error { return errors.New("no joy catalog found at \"catalogDir\"") },
				Git: GitOpts{
					IsValid:               func(string) bool { return true },
					GetUncommittedChanges: func(string) ([]string, error) { return nil, nil },
					GetCurrentCommit:      func(string) (string, error) { return "def456", nil },
					IsRemoteBranch:        func(string, string) (bool, error) { return false, nil },
					ResolveCommit:         func(string, string) (string, error) { return "abc123", nil },
				},
			},
			Expected: Group{
				Title: "Catalog",
				SubGroups: Groups{
					{
						Title: "Git working copy", Messages: Messages{
							{Type: "info", Value: "Directory exists: catalogDir"},
							{Type: "success", Value: "Working copy is valid"},
							{Type: "success", Value: "Working copy has no uncommitted changes"},
							{
								Type:  "warning",
								Value: "Catalog ref v1 should be checked out (instead of def456)",
								Details: Messages{
									{Type: "hint", Value: "Switch to catalog ref using: joy git checkout abc123"},
								},
							},
							{Type: "info", Value: "Current commit: def456"},
						},
					},
					{
						Title: "Loading catalog", Messages: Messages{
							{Type: "failed", Value: "Catalog not detected: no joy catalog found at \"catalogDir\""},
						},
					},
				}, topLevel: true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := config.ToFlagsContext(context.Background(), &config.GlobalFlags{CatalogRef: tc.CatalogRef})
			require.Equal(t, tc.Expected, diagnoseCatalog(ctx, &config.Config{User: config.User{CatalogDir: "catalogDir"}}, tc.Opts).StripAnsi())
		})
	}
}
