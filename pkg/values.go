package joy

import (
	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/release/render"
	"github.com/nestoca/joy/internal/yml"
)

type (
	IO = internal.IO
)

var ComputeReleaseValues = render.HydrateValues

type YAMLFile = yml.File
