package project

import (
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"

	"github.com/nestoca/joy/pkg/catalog"
)

func List(catalogDir string) error {
	cat, err := catalog.Load(catalog.LoadOpts{Dir: catalogDir})
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	// Configure table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderLine(false)
	table.SetAutoWrapText(true)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetCenterSeparator("")

	// Add header
	headers := []string{"NAME", "OWNERS"}
	table.SetHeader(headers)

	// Add rows
	for _, proj := range cat.Projects {
		owners := strings.Join(proj.Spec.Owners, " ")
		row := []string{proj.Name, owners}
		table.Append(row)
	}

	table.Render()
	return nil
}
