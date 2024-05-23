package joy

import (
	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/release/render"
)

type (
	ValueMapping = config.ValueMapping
	IO           = internal.IO
)

var ComputeReleaseValues = render.HydrateValues
