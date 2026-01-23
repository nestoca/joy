package joy

import (
	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/release/render"
	"github.com/nestoca/joy/internal/yml"
)

type (
	IO           = internal.IO
	RenderParams = render.RenderParams
)

var (
	ComputeReleaseValues = render.HydrateValues
	Render               = render.Render
)

type YAMLFile = yml.File
