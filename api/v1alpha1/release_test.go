package v1alpha1_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/api/v1alpha1"
)

func TestReleaseUnmarshalling(t *testing.T) {
	data := `{
	  kind: Release,
	  spec: {
			autoSync: false,
			values: {
				str: !lock string, 
				yes: !lock true,
				nil: !lock null,
				age: !lock 42,
			},
	  },
	}`

	var release v1alpha1.Release
	require.NoError(t, yaml.Unmarshal([]byte(data), &release))
	require.Equal(t, "string", release.Spec.Values["str"])
	require.Equal(t, true, release.Spec.Values["yes"])
	require.Equal(t, nil, release.Spec.Values["nil"])
	require.Equal(t, 42, release.Spec.Values["age"])

	require.NotNil(t, release.Spec.AutoSync)
	require.Equal(t, false, *release.Spec.AutoSync)
}

func TestReleaseChartValidation(t *testing.T) {
	cases := []struct {
		Name      string
		Chart     v1alpha1.ReleaseChart
		ValidRefs []string
		Error     string
	}{
		{
			Name: "only repoUrl",
			Chart: v1alpha1.ReleaseChart{
				RepoUrl: "test",
			},
			Error: "repoUrl and name must be defined together",
		},
		{
			Name: "only name",
			Chart: v1alpha1.ReleaseChart{
				Ref:  "r",
				Name: "test",
			},
			Error: "repoUrl and name must be defined together",
		},
		{
			Name: "both ref and repo at once",
			Chart: v1alpha1.ReleaseChart{
				Ref:     "ref",
				RepoUrl: "repo",
				Name:    "test",
				Version: "v1",
			},
			Error: "ref and repoUrl cannot both be present",
		},
		{
			Name: "repoUrl and name without version",
			Chart: v1alpha1.ReleaseChart{
				RepoUrl: "test",
				Name:    "test",
			},
			Error: "version is required when chart is not a reference",
		},
		{
			Name: "version not required with file uris",
			Chart: v1alpha1.ReleaseChart{
				RepoUrl: "file:///test",
				Name:    "chart",
			},
		},
		{
			Name: "invalid ref",
			Chart: v1alpha1.ReleaseChart{
				Ref: "ref",
			},
			Error: "unknown ref: ref",
		},
		{
			Name: "valid full chart def",
			Chart: v1alpha1.ReleaseChart{
				RepoUrl: "repo",
				Name:    "name",
				Version: "version",
			},
		},
		{
			Name: "valid ref without version",
			Chart: v1alpha1.ReleaseChart{
				Ref: "ref",
			},
			ValidRefs: []string{"ref"},
		},
		{
			Name: "valid ref with version",
			Chart: v1alpha1.ReleaseChart{
				Ref:     "ref",
				Version: "version",
			},
			ValidRefs: []string{"ref"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Error == "" {
				require.NoError(t, tc.Chart.Validate(tc.ValidRefs))
				return
			}
			require.EqualError(t, tc.Chart.Validate(tc.ValidRefs), tc.Error)
		})
	}
}
