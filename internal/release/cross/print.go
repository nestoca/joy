package cross

import (
	"github.com/olekukonko/tablewriter"
	"os"
	"sort"
	"strings"
)

func (r *ReleaseList) Print() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderLine(false)
	table.SetAutoWrapText(true)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetCenterSeparator("")

	headers := []string{"NAME"}
	for _, env := range r.Environments {
		headers = append(headers, strings.ToUpper(env.Name))
	}
	table.SetHeader(headers)

	// Sort releases by name
	sortedReleaseNames := make([]string, 0, len(r.Releases))
	for releaseName := range r.Releases {
		sortedReleaseNames = append(sortedReleaseNames, releaseName)
	}
	sort.Strings(sortedReleaseNames)

	for _, releaseName := range sortedReleaseNames {
		release := r.Releases[releaseName]
		row := []string{release.Name}
		for _, rel := range release.Releases {
			if rel != nil {
				row = append(row, rel.Spec.Version)
			} else {
				row = append(row, "-")
			}
		}
		table.Append(row)
	}

	table.Render()
}
