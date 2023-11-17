package diagnostics

import (
	"fmt"
	"os"

	"github.com/nestoca/joy/internal/config"
)

func diagnoseConfig(cfg *config.Config) (group Group) {
	group.Title = "Config"
	group.toplevel = true

	if _, err := os.Stat(cfg.FilePath); os.IsNotExist(err) {
		group.AddMsg(failed, label("File does not exist", cfg.FilePath))
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
