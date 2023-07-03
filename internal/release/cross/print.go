package cross

import (
	"github.com/TwiN/go-color"
	"github.com/olekukonko/tablewriter"
	"os"
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

	for _, release := range r.SortedReleases() {
		// Check if releases and their values are synced across all environments
		releasesSynced := release.AllReleasesSynced()
		valuesSynced := release.AllValuesSynced()

		row := []string{colorize(release.Name, releasesSynced, valuesSynced)}
		for _, rel := range release.Releases {
			text := "-"
			if rel != nil {
				text = rel.Spec.Version
			}
			text = colorize(text, releasesSynced, valuesSynced)
			row = append(row, text)
		}
		table.Append(row)
	}

	table.Render()
}

func colorize(text string, releasesSynced, valuesSynced bool) string {
	if !releasesSynced || !valuesSynced {
		if !releasesSynced {
			return color.InRed(text)
		} else {
			return color.InYellow(text)
		}
	}
	return color.InGreen(text)
}
