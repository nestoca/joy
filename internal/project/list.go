package project

import (
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"

	"github.com/nestoca/joy/pkg/catalog"
)

func List(cat *catalog.Catalog) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Configure(func(cfg *tablewriter.Config) {
		cfg.Header.Alignment = tw.CellAlignment{Global: tw.AlignLeft}
	})
	headers := []string{"NAME", "OWNERS"}
	table.Header(headers)

	for _, proj := range cat.Projects {
		owners := strings.Join(proj.Spec.Owners, " ")
		_ = table.Append([]string{proj.Name, owners})
	}

	_ = table.Render()
	return nil
}
