package v1alpha1_test

import (
	"testing"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestReleaseUnmarshalling(t *testing.T) {
	data := `{
	  kind: Release,
	  spec: {
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
}
