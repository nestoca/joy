package promote

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/kyokomi/emoji"
	"github.com/nestoca/joy-cli/internal/utils"
	"gopkg.in/yaml.v3"
)

type Opts struct {
	Environment string
	Project     string
	Version     string
	CatalogDir  string
}

func Promote(opts Opts) error {
	envReleasesDir := filepath.Join(opts.CatalogDir, "environments", opts.Environment, "releases")

	type promoteTarget struct {
		File    os.FileInfo
		Path    string
		Release *yaml.Node
	}
	var targets []*promoteTarget

	err := filepath.Walk(envReleasesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".release.yaml") {
			return nil
		}

		file, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading release file: %w", err)
		}

		release := &yaml.Node{}
		err = yaml.Unmarshal(file, release)
		if err != nil {
			return fmt.Errorf("parsing release file: %w", err)
		}

		releaseProject, err := utils.FindNode(release, ".spec.project")
		if err != nil {
			return fmt.Errorf("reading release's project: %w", err)
		}

		if releaseProject != nil && opts.Project == releaseProject.Value {
			targets = append(targets, &promoteTarget{
				File:    info,
				Release: release,
				Path:    path,
			})
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking catalog directory: %w", err)
	}

	for _, target := range targets {
		versionNode, err := utils.FindNode(target.Release, ".spec.version")
		if err != nil {
			return fmt.Errorf("updating release version: %w", err)
		}
		versionNode.Value = opts.Version

		result, err := utils.EncodeYaml(target.Release)
		if err != nil {
			return fmt.Errorf("encoding updated release: %w", err)
		}
		err = os.WriteFile(target.Path, result, target.File.Mode())
		if err != nil {
			return fmt.Errorf("writing to release file: %w", err)
		}

		releaseName, err := utils.FindNode(target.Release, ".metadata.name")
		if err != nil {
			return fmt.Errorf("reading release's name: %w", err)
		}

		_, _ = emoji.Printf(":check_mark:Promoted release %s to version %s\n", color.HiBlueString(releaseName.Value), color.GreenString(opts.Version))
	}

	if len(targets) == 0 {
		return errors.New(emoji.Sprintf(":warning:Did not find any releases for project %s\n", color.HiYellowString(opts.Project)))
	}

	_, _ = emoji.Printf("\n:beer:Done! Promoted releases of project %s in environment %s to version %s\n", color.HiCyanString(opts.Project), color.HiCyanString(opts.Environment), color.GreenString(opts.Version))

	return nil
}