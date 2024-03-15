package joy

import (
	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/release/render"
)

type ValueMapping = config.ValueMapping

// ReleaseValues returns the values from a Release.Spec.Values after all chart mappings have been applied and templated values subsituted.
func ReleaseValues(release *v1alpha1.Release, mappings *ValueMapping) (map[string]any, error) {
	return render.HydrateValues(release, mappings)
}
