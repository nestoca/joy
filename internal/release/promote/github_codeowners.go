package promote

import (
	"os"
	"strings"
)

func GetCodeOwners(dir string) ([]string, error) {
	var owners []string
	for _, relativeFilepath := range []string{".github/CODEOWNERS", "CODEOWNERS"} {
		filepath := dir + "/" + relativeFilepath
		_, err := os.Stat(filepath)
		if err != nil {
			continue
		}
		content, err := os.ReadFile(filepath)
		if err != nil {
			return nil, err
		}
		owners = append(owners, parseCodeOwners(string(content))...)
	}
	return owners, nil
}

func parseCodeOwners(content string) []string {
	var owners []string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, " ")
		if len(parts) < 2 {
			continue
		}

		for _, owner := range parts[1:] {
			if owner == "" {
				continue
			}
			owners = append(owners, strings.TrimLeft(owner, "@"))
		}
	}

	return owners
}
