package config

import "context"

type GlobalFlags struct {
	// SkipCatalogUpdate global flag used to skip catalog update and dirty check.
	SkipCatalogUpdate bool
}

type flagKey struct{}

func ToFlagsContext(parent context.Context, flags *GlobalFlags) context.Context {
	return context.WithValue(parent, flagKey{}, flags)
}

func FlagsFromContext(ctx context.Context) *GlobalFlags {
	cfg, _ := ctx.Value(flagKey{}).(*GlobalFlags)
	return cfg
}
