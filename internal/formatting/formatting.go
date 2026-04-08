package formatting

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

type Format string

const (
	FormatTable    Format = "table"
	FormatJson     Format = "json"
	FormatYaml     Format = "yaml"
	FormatNames    Format = "names"
	FormatRelPaths Format = "rel-paths"
	FormatAbsPaths Format = "abs-paths"
)

func RenderJson(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func RenderYaml(writer io.Writer, value any) error {
	encoder := yaml.NewEncoder(writer)
	encoder.SetIndent(2)
	return encoder.Encode(value)
}

func RenderNames[T Named](writer io.Writer, items []T) error {
	for _, item := range items {
		if _, err := fmt.Fprintln(writer, item.GetName()); err != nil {
			return err
		}
	}
	return nil
}

// PathOutputLines converts absolute file paths to either absolute or catalog-relative
// paths, deduplicates, and returns them sorted. catalogDir must be the absolute
// catalog root when useAbs is false.
func PathOutputLines(catalogDir string, useAbs bool, absFilePaths []string) ([]string, error) {
	uniq := make(map[string]struct{})
	for _, p := range absFilePaths {
		if p == "" {
			continue
		}
		uniq[p] = struct{}{}
	}
	out := make([]string, 0, len(uniq))
	for p := range uniq {
		var line string
		var err error
		if useAbs {
			line = p
		} else {
			if catalogDir == "" {
				return nil, fmt.Errorf("catalog directory is required for rel-paths output")
			}
			line, err = filepath.Rel(catalogDir, p)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", p, err)
			}
		}
		out = append(out, line)
	}
	slices.Sort(out)
	return out, nil
}

// RenderPathFormat writes one path per line (sorted, deduplicated).
func RenderPathFormat(writer io.Writer, catalogDir string, useAbs bool, absFilePaths []string) error {
	lines, err := PathOutputLines(catalogDir, useAbs, absFilePaths)
	if err != nil {
		return err
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(writer, line); err != nil {
			return err
		}
	}
	return nil
}

// AddFormatFlag registers --format / -f for render format (table, json, yaml, names, rel-paths, abs-paths).
func AddFormatFlag(cmd *cobra.Command, format *Format) {
	cmd.Flags().StringVarP((*string)(format), "format", "f", string(FormatTable), "output format, one of: table, json, yaml, names, rel-paths, abs-paths")
}
