package environment

import (
	"fmt"
	"io"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/nestoca/joy/internal/output"
	"github.com/nestoca/joy/pkg/catalog"
)

func Render(cat *catalog.Catalog, writer io.Writer, format output.Format) error {
	switch format {
	case output.FormatJson:
		return output.RenderJson(writer, cat.Environments)
	case output.FormatYaml:
		return output.RenderYaml(writer, cat.Environments)
	case output.FormatNames:
		return output.RenderNames(writer, cat.Environments)
	case output.FormatTable:
		return renderTable(cat, writer)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
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
