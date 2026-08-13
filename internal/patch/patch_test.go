package patch

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type Resource struct {
	Spec Spec `json:"spec"`
}

type Spec struct {
	Namespace string `json:"namespace"`
	Values    Values `json:"values"`
}

type Values struct {
	Image    string            `json:"image"`
	Env      map[string]string `json:"env"`
	Frontend Frontend          `json:"frontend"`
}

type Frontend struct {
	Gateway *Gateway `json:"gateway"`
}

type Gateway struct {
	HttpRoutes HttpRoutes `json:"httpRoutes"`
}

type HttpRoutes struct{}

func base() *Resource {
	return &Resource{
		Spec: Spec{
			Namespace: "default",
			Values: Values{
				Env: map[string]string{
					"ZULU":  "z",
					"ALPHA": "a",
				},
				Frontend: Frontend{
					Gateway: &Gateway{
						HttpRoutes: HttpRoutes{},
					},
				},
			},
		},
	}
}

func TestApply(t *testing.T) {
	out, err := Apply(
		base(),
		[]Op{
			{Op: "add", Path: "/spec/values/image", Value: "backoffice"},
			{Op: "replace", Path: "/spec/values/env/ALPHA", Value: "A2"},
			{Op: "remove", Path: "/spec/values/frontend/gateway"},
		},
	)
	require.NoError(t, err)
	require.Equal(t, "z", out.Spec.Values.Env["ZULU"])
	require.Equal(t, "A2", out.Spec.Values.Env["ALPHA"])
	require.Equal(t, "backoffice", out.Spec.Values.Image)
	require.Nil(t, out.Spec.Values.Frontend.Gateway)
}

func TestApply_NoOpsReturnsInput(t *testing.T) {
	in := base()
	out, err := Apply(in, nil)
	require.NoError(t, err)
	require.Same(t, in, out)
}

func TestApply_StrictErrors(t *testing.T) {
	// RFC 6902: replacing a non-existent path is an error.
	_, err := Apply(base(), []Op{{Op: "replace", Path: "/spec/values/nope", Value: "x"}})
	require.Error(t, err)
	// add to a non-existent parent is an error.
	_, err = Apply(base(), []Op{{Op: "add", Path: "/spec/values/image/name", Value: "x"}})
	require.Error(t, err)
}
