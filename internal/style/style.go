package style

import "github.com/TwiN/go-color"

const (
	darkGrey   = "\033[38;2;90;90;90m"
	darkYellow = "\033[38;2;128;128;0m"
)

// SecondaryInfo is for text that should be less prominent than the main text
func SecondaryInfo(s any) string {
	return color.Colorize(darkGrey, s)
}

// Resource is for the name of resources and entities such as releases, projects, environments, clusters, etc.
func Resource(s any) string {
	return color.InBold(color.InYellow(s))
}

func Author(s any) string {
	return color.InBold(color.InCyan(s))
}

func OutOfSyncRelease(s any) string {
	return color.InYellow(s)
}

func InSyncRelease(s any) string {
	return color.InGray(s)
}

func InSyncReleaseVersion(s any) string {
	return color.InGray(s)
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
	return color.InBold(color.InYellow(s))
}

// Version is for release versions within messages (not tables)
func Version(s any) string {
	return color.InBold(color.InGreen(s))
}

// ReleaseInSync is for releases that are in sync across environments
func ReleaseInSync(s any) string {
	return color.InGreen(s)
}

// ReleaseOutOfSync is for releases that are not in sync across environments
func ReleaseOutOfSync(s any) string {
	return color.InRed(s)
}

// ReleaseNotAvailable is for releases that are not existing in an environment or have not version specified
func ReleaseNotAvailable(s any) string {
	return color.InGray(s)
}

// Link is for URLs and other links
func Link(s any) string {
	return color.InUnderline(color.InBlue(s))
}

// DiagnosticHeader is for the header of a diagnostic
func DiagnosticHeader(s any) string {
	return color.InBold(color.InWhite(s))
}

// DiagnosticGroup is for the header of a diagnostic section
func DiagnosticGroup(s any) string {
	return color.InBold(color.InBlue(s))
}

// DiagnosticLabel is for the label of a diagnostic item
func DiagnosticLabel(s any) string {
	return color.InBold(s)
}

// ResourceKind is for a kind of resource (eg: `Release`, `Project`, `Environment`)
func ResourceKind(s any) string {
	return color.InBold(s)
}

func InSyncVersion(s any) string {
	return color.InGreen(s)
}

func BehindVersion(s any) string {
	return color.InYellow(s)
}

func AheadVersion(s any) string {
	return color.InPurple(s)
}

func DirtyVersion(s any) string {
	return color.InRed(s)
}

func Notice(s any) string {
	return color.InBold(color.InPurple(s))
}
