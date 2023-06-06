package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/kyokomi/emoji"
	"github.com/nestoca/joy-cli/internal/utils"
	"github.com/spf13/viper"

	"gopkg.in/yaml.v3"
)

type PromoteArgs struct{
	Environment string
	Project string
	Version string
}

type promoteTarget struct {
	File os.FileInfo
	Path string
	Release interface{}
}

func Promote(args PromoteArgs) error {
	catalogDir, err := utils.ResolvePath(viper.GetString("catalogDir"))
	if err != nil {
		return fmt.Errorf("failed to resolve catalog directory path: %w", err)
	}

	resolvedCatalogDir := filepath.Join(catalogDir, "environments", args.Environment, "releases")

	var targets []*promoteTarget

	err = filepath.Walk(resolvedCatalogDir, func(path string, info os.FileInfo, err error) error {
		if (err != nil) {
			return err
		}
		
		if info.IsDir() || !(strings.HasSuffix(info.Name(), ".release.yaml") || strings.HasSuffix(info.Name(), ".release.yml")) {
			return nil
		}

		file, err := os.ReadFile(path)
		if (err != nil) {
			return fmt.Errorf("could not read release file: %w", err)
		}

		var release interface{}
		err = yaml.Unmarshal(file, &release)
		if (err != nil) {
			return fmt.Errorf("could not parse release file: %w", err)
		}

		releaseProject, err := utils.TraverseYAML(release, ".spec.project")
		if (err != nil) {
			return fmt.Errorf("could not traverse release yaml: %w", err)
		}

		if (releaseProject != nil && args.Project == releaseProject.(string)){
			targets = append(targets, &promoteTarget{
				File: info,
				Release: release,
				Path: path,
			})
		}
		return nil
	})
	if (err != nil) {
		return fmt.Errorf("failed to walk catalog directory: %w", err)
	}

	for _, target := range targets {
		err := utils.SetYAMLValue(target.Release, ".spec.version", args.Version)
		if (err != nil) {
			return fmt.Errorf("failed to marshal updated release: %w", err)
		}
	
		result, err := yaml.Marshal(target.Release)
		if (err != nil) {
			return fmt.Errorf("failed to marshal updated release: %w", err)
		}
		err = os.WriteFile(target.Path, result, target.File.Mode())
		if (err != nil) {
			return fmt.Errorf("failed to write updated release file: %w", err)
		}

		releaseName, err := utils.TraverseYAML(target.Release, ".metadata.name")
		if (err != nil) {
			return fmt.Errorf("could not traverse release yaml: %w", err)
		}

		emoji.Printf(":check_mark: Promoted release %s to %s\n", color.HiBlueString(releaseName.(string)), color.GreenString(args.Version))
	}

	if (len(targets) > 0) {
		fmt.Println("")
		emoji.Printf(":beer: Done! Promoted all releases in %s for project %s\n", color.HiCyanString(args.Environment), color.HiCyanString(args.Project))
	} else {
		emoji.Printf(":warning: Did not find any releases for project %s\n", color.HiYellowString(args.Project))
	}

	return nil
}