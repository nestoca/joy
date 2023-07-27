package style

import "github.com/TwiN/go-color"

const darkGrey = "\033[38;2;90;90;90m"
const darkYellow = "\033[38;2;128;128;0m"

// SecondaryInfo is for text that should be less prominent than the main text
func SecondaryInfo(s any) string {
	return color.Colorize(darkGrey, s)
}

// Resource is for the name of resources and entities such as releases, projects, environments, clusters, etc.
func Resource(s any) string {
	return color.InBold(color.InYellow(s))
}

// ResourceEnvPrefix is for the environment prefix in front of a resource name (eg: `staging` in `staging/<release-name>`)
func ResourceEnvPrefix(s any) string {
	return color.Colorize(darkYellow, s)
}

// DiffBefore is for text that is being removed in a diff
func DiffBefore(s any) string {
	return color.InRed(s)
}

// DiffAfter is for text that is being added in a diff
func DiffAfter(s any) string {
	return color.InGreen(s)
}

// OK is for items that are in expected state, such as a release that is in sync with other environments
func OK(s any) string {
	return color.InGreen(s)
}

// Warning is for text that is a warning or an error, such as a missing/unsynched release or an uncommitted change
func Warning(s any) string {
	return color.InRed(s)
}

// Code is for code snippets, commands, yaml properties, or any technical text that is not a resource name
func Code(s any) string {
	return color.InBold(color.InCyan(s))
}

// Version is for release versions within messages (not tables)
func Version(s any) string {
	return color.InBold(color.InGreen(s))
}
