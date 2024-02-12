package joy

import (
	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/release/render"
)

// ReleaseValues returns the values from a Release.Spec.Values after all chart mappings have been applied and templated values subsituted.
func ReleaseValues(release *v1alpha1.Release, environment *v1alpha1.Environment, mappings map[string]any) (map[string]any, error) {
	return render.HydrateValues(release, environment, mappings)
}
