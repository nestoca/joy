package project

import (
	"fmt"
	"github.com/nestoca/joy/internal/git"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"

	"github.com/nestoca/joy/pkg/catalog"
)

func List(catalogDir string, skipCatalogUpdate bool) error {
	if skipCatalogUpdate {
		fmt.Println("ℹ️ Skipping catalog update and dirty check.")
	} else {
		if err := git.EnsureCleanAndUpToDateWorkingCopy(catalogDir); err != nil {
			return err
		}
	}

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
