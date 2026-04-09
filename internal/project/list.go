package project

import (
	"fmt"
	"io"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/nestoca/joy/internal/formatting"
	"github.com/nestoca/joy/pkg/catalog"
)

func Render(cat *catalog.Catalog, writer io.Writer, format formatting.Format) error {
	switch format {
	case formatting.FormatJson:
		if err := addPathsToProjects(cat); err != nil {
			return fmt.Errorf("adding paths to projects: %w", err)
		}
		return formatting.RenderJson(writer, cat.Projects)
	case formatting.FormatYaml:
		if err := addPathsToProjects(cat); err != nil {
			return fmt.Errorf("adding paths to projects: %w", err)
		}
		return formatting.RenderYaml(writer, cat.Projects)
	case formatting.FormatNames:
		return formatting.RenderNames(writer, cat.Projects)
	case formatting.FormatRelPaths:
		return formatting.RenderRelativePaths(writer, projectFilePaths(cat), cat.Dir)
	case formatting.FormatAbsPaths:
		return formatting.RenderAbsolutePaths(writer, projectFilePaths(cat))
	case formatting.FormatTable:
		return renderTable(cat, writer)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func addPathsToProjects(cat *catalog.Catalog) error {
	for _, proj := range cat.Projects {
		if proj == nil || proj.File == nil {
			continue
		}
		rel, abs, err := formatting.GetRelativeAndAbsolutePaths(proj.File.Path, cat.Dir)
		if err != nil {
			return err
		}
		proj.RelativePath = rel
		proj.AbsolutePath = abs
	}
	return nil
}

func projectFilePaths(cat *catalog.Catalog) []string {
	var paths []string
	for _, proj := range cat.Projects {
		if proj == nil || proj.File == nil {
			continue
		}
		paths = append(paths, proj.File.Path)
	}
	return paths
}

func renderTable(cat *catalog.Catalog, writer io.Writer) error {
	t := table.NewWriter()
	t.SetStyle(table.StyleRounded)

	t.AppendHeader(table.Row{"NAME", "OWNERS"})

	for _, proj := range cat.Projects {
		owners := strings.Join(proj.Spec.Owners, " ")
		t.AppendRow(table.Row{proj.Name, owners})
	}

	rendered := t.Render()
	_, err := io.WriteString(writer, rendered+"\n")
	if err != nil {
		return fmt.Errorf("writing project list as table: %w", err)
	}
	return nil
}
