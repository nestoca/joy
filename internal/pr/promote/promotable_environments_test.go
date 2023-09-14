package promote

import (
	"github.com/go-test/deep"
	"github.com/nestoca/joy/api/v1alpha1"
	"testing"
)

func newEnvironment(name string, promotable bool) *v1alpha1.Environment {
	return &v1alpha1.Environment{
		EnvironmentMetadata: v1alpha1.EnvironmentMetadata{
			Name: name,
		},
		Spec: v1alpha1.EnvironmentSpec{
			Promotion: v1alpha1.Promotion{
				FromPullRequests: promotable,
			},
		},
	}
}

func NewEnvironments() []*v1alpha1.Environment {
	return []*v1alpha1.Environment{
		newEnvironment("staging", true),
		newEnvironment("qa", false),
		newEnvironment("production", false),
		newEnvironment("demo", true),
	}
}

func TestGetPromotableEnvironmentNames(t *testing.T) {
	cases := []struct {
		name          string
		environments  []*v1alpha1.Environment
		expectedNames []string
	}{
		{
			name:          "no environments",
			environments:  []*v1alpha1.Environment{},
			expectedNames: nil,
		},
		{
			name:          "some promotable environments",
			environments:  NewEnvironments(),
			expectedNames: []string{"staging", "demo"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			names := getPromotableEnvironmentNames(c.environments)
			if diff := deep.Equal(c.expectedNames, names); diff != nil {
				t.Error(diff)
			}
		})
	}
}
