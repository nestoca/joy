package environment

import (
	"fmt"
	"io"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/formatting"
	"github.com/nestoca/joy/pkg/catalog"
)

func Render(cat *catalog.Catalog, writer io.Writer, format formatting.Format) error {
	switch format {
	case formatting.FormatJson:
		err := addPathsToEnvironments(cat.Environments, cat.Dir)
		if err != nil {
			return fmt.Errorf("adding paths to environments: %w", err)
		}
		return formatting.RenderJson(writer, cat.Environments)
	case formatting.FormatYaml:
		err := addPathsToEnvironments(cat.Environments, cat.Dir)
		if err != nil {
			return fmt.Errorf("adding paths to environments: %w", err)
		}
		return formatting.RenderYaml(writer, cat.Environments)
	case formatting.FormatNames:
		return formatting.RenderNames(writer, cat.Environments)
	case formatting.FormatRelPaths:
		return formatting.RenderRelativePaths(writer, environmentFilePaths(cat.Environments), cat.Dir)
	case formatting.FormatAbsPaths:
		return formatting.RenderAbsolutePaths(writer, environmentFilePaths(cat.Environments))
	case formatting.FormatTable:
		return renderTable(cat, writer)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func addPathsToEnvironments(envs []*v1alpha1.Environment, catalogDir string) error {
	for _, env := range envs {
		if env == nil || env.File == nil {
			continue
		}
		rel, abs, err := formatting.GetRelativeAndAbsolutePaths(env.File.Path, catalogDir)
		if err != nil {
			return err
		}
		env.RelativePath = rel
		env.AbsolutePath = abs
	}
	return nil
}

func environmentFilePaths(envs []*v1alpha1.Environment) []string {
	var paths []string
	for _, env := range envs {
		if env == nil || env.File == nil {
			continue
		}
		paths = append(paths, env.File.Path)
	}
	return paths
}

func renderTable(cat *catalog.Catalog, writer io.Writer) error {
	t := table.NewWriter()
	t.SetStyle(table.StyleRounded)

	t.AppendHeader(table.Row{"NAME", "OWNERS"})

	for _, env := range cat.Environments {
		owners := strings.Join(env.Spec.Owners, " ")
		t.AppendRow(table.Row{env.Name, owners})
	}

	rendered := t.Render()
	_, err := io.WriteString(writer, rendered+"\n")
	if err != nil {
		return fmt.Errorf("writing environment list as table: %w", err)
	}
	return nil
}
