package formatting

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

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

// GetRelativeAndAbsolutePaths returns catalog-relative and clean absolute paths for given absolute catalog resource file path.
func GetRelativeAndAbsolutePaths(absFilePath, catalogDir string) (relativePath, absolutePath string, err error) {
	if absFilePath == "" {
		return "", "", nil
	}
	if catalogDir == "" {
		return "", "", fmt.Errorf("catalog directory is required for list metadata paths")
	}
	rel, err := filepath.Rel(catalogDir, absFilePath)
	if err != nil {
		return "", "", err
	}
	return rel, filepath.Clean(absFilePath), nil
}

func RenderNames[T Named](writer io.Writer, items []T) error {
	for _, item := range items {
		if _, err := fmt.Fprintln(writer, item.GetName()); err != nil {
			return err
		}
	}
	return nil
}

// RenderRelativePaths writes all paths as catalog-relative paths on different lines.
func RenderRelativePaths(writer io.Writer, absFilePaths []string, catalogDir string) error {
	for _, path := range absFilePaths {
		rel, err := filepath.Rel(catalogDir, path)
		if err != nil {
			return fmt.Errorf("getting relative path for %s: %w", path, err)
		}
		if _, err := fmt.Fprintln(writer, rel); err != nil {
			return err
		}
	}
	return nil
}

// RenderAbsolutePaths writes all paths as absolute paths on different lines.
func RenderAbsolutePaths(writer io.Writer, absFilePaths []string) error {
	for _, path := range absFilePaths {
		if _, err := fmt.Fprintln(writer, path); err != nil {
			return err
		}
	}
	return nil
}

// AddFormatFlag registers --format / -f for render format (table, json, yaml, names, rel-paths, abs-paths).
func AddFormatFlag(cmd *cobra.Command, format *Format) {
	cmd.Flags().StringVarP((*string)(format), "format", "f", string(FormatTable), "output format, one of: table, json, yaml, names, rel-paths, abs-paths")
}
