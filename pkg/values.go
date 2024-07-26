package joy

import (
	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/release/render"
)

type (
	IO = internal.IO
)

var ComputeReleaseValues = render.HydrateValues
