package build

import (
	"testing"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/release/cross"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
	"github.com/stretchr/testify/require"
)

func TestPromote(t *testing.T) {
	var writer yml.WriterMock

	opts := Opts{
		Catalog: &catalog.Catalog{
			Environments: []*v1alpha1.Environment{{}},
			Releases: cross.ReleaseList{
				Items: []*cross.Release{
					{
						Releases: []*v1alpha1.Release{
							{
								Spec: v1alpha1.ReleaseSpec{Project: "promote-build"},
								File: makeFile(t, "{ spec: { version: 0.0.0 } }"),
							},
						},
					},
				},
			},
		},
		Writer:  &writer,
		Project: "promote-build",
		Version: "1.1.2",
	}

	require.NoError(t, Promote(opts))
	require.Len(t, writer.WriteFileCalls(), 1)

	file := writer.WriteFileCalls()[0].File
	require.Equal(t, "1.1.2", yml.FindNodeValueOrDefault(file.Tree, "spec.version", ""))
}

func TestPromoteWithInvalidVersion(t *testing.T) {
	opts := Opts{
		Catalog: &catalog.Catalog{
			Environments: []*v1alpha1.Environment{{}},
		},
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.1.2-updated",
	}
	require.EqualError(t, Promote(opts), "cannot promote release with non-standard version to testing environment")
}

func TestPromoteWhenNoReleasesFoundForProject(t *testing.T) {
	opts := Opts{
		Catalog: &catalog.Catalog{
			Environments: []*v1alpha1.Environment{{}},
			Releases:     cross.ReleaseList{},
		},
		Environment: "testing",
		Project:     "promote-build",
		Version:     "1.1.2",
	}

	require.EqualError(t, Promote(opts), "no releases found for project promote-build")
}

func makeFile(t *testing.T, content string) *yml.File {
	t.Helper()
	f, err := yml.NewFile("", []byte(content))
	require.NoError(t, err)
	return f
}
