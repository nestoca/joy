package environment

import (
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"

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
	table := tablewriter.NewWriter(writer)
	table.Configure(func(cfg *tablewriter.Config) {
		cfg.Header.Alignment = tw.CellAlignment{Global: tw.AlignLeft}
	})
	headers := []string{"NAME", "OWNERS"}
	table.Header(headers)

	for _, env := range cat.Environments {
		owners := strings.Join(env.Spec.Owners, " ")
		_ = table.Append([]string{env.Name, owners})
	}

	return table.Render()
}
