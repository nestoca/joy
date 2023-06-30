package promote

import "github.com/nestoca/joy-cli/internal/release"

type Promotion struct {
	Release release.Release
	Values  release.Values

	// Is the release version being promoted?
	IsVersionPromoted bool

	// Is the rest of the release or values being promoted?
	IsMorePromoted bool
}

type List struct {
	SourceEnv  string
	TargetEnv  string
	Promotions []Promotion
}
