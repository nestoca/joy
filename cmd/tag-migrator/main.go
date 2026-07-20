package main

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/nestoca/joy/internal/yml"
	joy "github.com/nestoca/joy/pkg"
	"gopkg.in/yaml.v3"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	catalog, err := joy.LoadCatalog(context.Background(), "")
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	for _, cross := range catalog.Releases.Items {
		for _, release := range cross.Releases {
			if release == nil {
				continue
			}

			moveTags(release.File.Tree)

			data, err := release.File.ToYaml()
			if err != nil {
				return fmt.Errorf("failed to marshal data for release: %s/%s: %v", release.Environment.Name, release.Name, err)
			}

			if err := os.WriteFile(release.File.Path, []byte(data), 0o644); err != nil {
				return fmt.Errorf("failed to write data for release: %s/%s: %v", release.Environment.Name, release.Name, err)
			}
		}
	}

	return nil
}

func moveTags(node *yaml.Node) {
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			var (
				key   = node.Content[i]
				value = node.Content[i+1]
			)
			if slices.Contains(yml.CustomTags, value.Tag) {
				key.Tag = value.Tag
				value.Tag = ""
			}
		}
	}
	for _, node := range node.Content {
		moveTags(node)
	}
}
