package joy

import (
	"github.com/nestoca/joy/internal/info"
	"github.com/nestoca/joy/internal/links"
)

type LinksProvider = links.Provider

func NewLinksProvider(cfg Config) LinksProvider {
	return links.NewProvider(
		info.NewProvider(
			cfg.GitHubOrganization,
			cfg.Templates.Project.GitTag,
			cfg.RepositoriesDir,
			cfg.JoyCache,
		),
		cfg.Templates,
	)
}
