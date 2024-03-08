package text

import (
	"strings"

	"github.com/TwiN/go-color"
	"github.com/pmezard/go-difflib/difflib"
)

type File struct {
	Name    string
	Content string
}

type DiffFunc func(actual, expected File, context int) string

func Diff(actual, expected File, context int) string {
	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        indentLines(difflib.SplitLines(expected.Content), "  "),
		B:        indentLines(difflib.SplitLines(actual.Content), "  "),
		FromFile: expected.Name,
		ToFile:   actual.Name,
		Context:  context,
	})
	return diff
}

func DiffColorized(actual, expected File, context int) string {
	return colorize(Diff(actual, expected, context))
}

func colorize(value string) string {
	lines := strings.Split(value, "\n")
	colorized := make([]string, len(lines))
	for i, line := range lines {
		if len(line) == 0 {
			continue
		}
		switch line[0] {
		case '-':
			colorized[i] = color.InRed(line)
		case '+':
			colorized[i] = color.InGreen(line)
		default:
			colorized[i] = line
		}
	}

	return strings.Join(colorized, "\n")
}

func indentLines(lines []string, indent string) []string {
	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = indent + line
	}
	return result
}
