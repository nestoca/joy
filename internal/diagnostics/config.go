package diagnostics

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/nestoca/joy/internal/config"
)

type ConfigOpts struct {
	Stat func(string) (fs.FileInfo, error)
}

func diagnoseConfig(cfg *config.Config, opts ConfigOpts) (group Group) {
	if opts.Stat == nil {
		opts.Stat = os.Stat
	}

	group.Title = "Config"
	group.topLevel = true

	if _, err := opts.Stat(cfg.FilePath); err != nil {
		if os.IsNotExist(err) {
			group.AddMsg(failed, label("File does not exist", cfg.FilePath))
			return
		}
		group.AddMsg(failed, fmt.Sprintf("Failed to get config file: %v", err))
		return
	}

	formatSelection := func(selected []string) string {
		if len(selected) == 0 {
			return "<all>"
		}
		return fmt.Sprintf("%d", len(selected))
	}

	group.
		AddMsg(success, label("File exists", cfg.FilePath)).
		AddMsg(info, label("Selected environments", formatSelection(cfg.Environments.Selected))).
		AddMsg(info, label("Selected releases", formatSelection(cfg.Releases.Selected)))

	return
}
