package config

import "context"

// DefaultCatalogRef is the git ref of the catalog used when none is specified.
const DefaultCatalogRef = "master"

type GlobalFlags struct {
	// SkipCatalogUpdate global flag used to skip catalog update and dirty check.
	SkipCatalogUpdate bool
	// CatalogRef global flag used to specify the git ref of the catalog to checkout.
	CatalogRef string
}

type flagKey struct{}

func ToFlagsContext(parent context.Context, flags *GlobalFlags) context.Context {
	return context.WithValue(parent, flagKey{}, flags)
}

func FlagsFromContext(ctx context.Context) *GlobalFlags {
	cfg, _ := ctx.Value(flagKey{}).(*GlobalFlags)
	return cfg
}

// CatalogRefFromContext returns the catalog git ref from the global flags in context, or DefaultCatalogRef if unset.
func CatalogRefFromContext(ctx context.Context) string {
	if flags := FlagsFromContext(ctx); flags != nil && flags.CatalogRef != "" {
		return flags.CatalogRef
	}
	return DefaultCatalogRef
}
