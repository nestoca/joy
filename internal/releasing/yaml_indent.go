package releasing

import (
	"strings"
	"unicode"
)

func getIndentSize(content string) int {
	lines := strings.Split(content, "\n")

	// Find the first non-empty line with indentation
	for _, line := range lines {
		trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)
		if len(trimmed) > 0 && line[0] == ' ' {
			indentSize := 0
			for _, ch := range line {
				if ch == ' ' {
					indentSize++
				} else {
					break
				}
			}
			return indentSize
		}
	}

	return 2 // Default indent size if no non-empty line with indentation is found
}
