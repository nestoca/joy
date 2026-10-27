package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCatalogRefFromContext(t *testing.T) {
	cases := []struct {
		Name     string
		Ctx      context.Context
		Expected string
	}{
		{
			Name:     "no flags in context",
			Ctx:      context.Background(),
			Expected: DefaultCatalogRef,
		},
		{
			Name:     "empty catalog ref",
			Ctx:      ToFlagsContext(context.Background(), &GlobalFlags{}),
			Expected: DefaultCatalogRef,
		},
		{
			Name:     "custom catalog ref",
			Ctx:      ToFlagsContext(context.Background(), &GlobalFlags{CatalogRef: "release/v1"}),
			Expected: "release/v1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			require.Equal(t, tc.Expected, CatalogRefFromContext(tc.Ctx))
		})
	}
}
