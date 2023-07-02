package cross

import (
	"github.com/TwiN/go-color"
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

		// Check if releases and their values are synced across all environments
		releasesSynced := release.AreReleasesSynced()
		valuesSynced := release.AreValuesSynced()

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
