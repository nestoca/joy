package project

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
		payload, err := projectsWithPaths(cat)
		if err != nil {
			return err
		}
		return formatting.RenderJson(writer, payload)
	case formatting.FormatYaml:
		payload, err := projectsWithPaths(cat)
		if err != nil {
			return err
		}
		return formatting.RenderYaml(writer, payload)
	case formatting.FormatNames:
		return formatting.RenderNames(writer, cat.Projects)
	case formatting.FormatRelPaths:
		return formatting.RenderPathFormat(writer, cat.Dir, false, projectFilePaths(cat))
	case formatting.FormatAbsPaths:
		return formatting.RenderPathFormat(writer, cat.Dir, true, projectFilePaths(cat))
	case formatting.FormatTable:
		return renderTable(cat, writer)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func projectsWithPaths(cat *catalog.Catalog) ([]*v1alpha1.Project, error) {
	out := make([]*v1alpha1.Project, len(cat.Projects))
	for i, proj := range cat.Projects {
		if proj == nil {
			continue
		}
		p := *proj
		p.File = nil
		if proj.File != nil {
			rel, abs, err := formatting.GetRelativeAndAbsolutePaths(proj.File.Path, cat.Dir)
			if err != nil {
				return nil, err
			}
			p.RelativePath = rel
			p.AbsolutePath = abs
		}
		out[i] = &p
	}
	return out, nil
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
