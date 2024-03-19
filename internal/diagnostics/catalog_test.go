package diagnostics

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/pkg/catalog"
)

func TestCatalogDiagnostics(t *testing.T) {
	cases := []struct {
		Name     string
		Catalog  *catalog.Catalog
		Opts     CatalogOpts
		Expected Group
	}{
		{
			Name:    "happy",
			Catalog: &catalog.Catalog{},
			Opts: CatalogOpts{
				Stat: func(string) (fs.FileInfo, error) { return nil, nil },
				Git: GitOpts{
					IsValid:               func(string) bool { return true },
					GetUncommittedChanges: func(string) ([]string, error) { return nil, nil },
					GetCurrentBranch:      func(string) (string, error) { return "master", nil },
					IsInSyncWithRemote:    func(string, string) (bool, error) { return true, nil },
					GetCurrentCommit:      func(string) (string, error) { return "origin/HEAD", nil },
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
							{Type: "success", Value: "Default branch master is checked out"},
							{Type: "success", Value: "Default branch is in sync with remote"},
							{Type: "info", Value: "Current commit: origin/HEAD"},
						},
					},
					{
						Title: "Resources", Messages: Messages{
							{Type: "info", Value: "Environments: 0"},
							{Type: "info", Value: "Projects: 0"},
							{Type: "info", Value: "Releases: 0"},
						},
					},
				},
				topLevel: true,
			},
		},

		{
			Name:    "git not valid repository",
			Catalog: &catalog.Catalog{},

			Opts: CatalogOpts{
				Stat: func(s string) (fs.FileInfo, error) { return nil, nil },
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
				}, topLevel: true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			actual := diagnoseCatalog("catalogDir", tc.Catalog, tc.Opts).StripAnsi()
			require.Equal(t, tc.Expected, actual)
		})
	}
}
