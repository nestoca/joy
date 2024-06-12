package ignore

import (
	"bufio"
	"os"
	"path"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

const (
	commentPrefix  = "#"
	ignoreFileName = ".joyignore"
)

type Matcher struct {
	giMatcher gitignore.Matcher
}

func (m *Matcher) Match(path string) bool {
	return m.giMatcher.Match(strings.Split(path, "/"), false)
}

func NewMatcher(catalogRootPath string) (*Matcher, error) {
	ignoreFilePath := path.Join(catalogRootPath, ignoreFileName)

	patterns, err := readIgnoreFile(ignoreFilePath)
	if err != nil {
		return nil, err
	}

	return &Matcher{
		gitignore.NewMatcher(patterns),
	}, nil
}

func readIgnoreFile(ignoreFilePath string) (patterns []gitignore.Pattern, err error) {
	f, err := os.Open(ignoreFilePath)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := scanner.Text()
		if !strings.HasPrefix(s, commentPrefix) && len(strings.TrimSpace(s)) > 0 {
			patterns = append(patterns, gitignore.ParsePattern(s, nil))
		}
	}

	return patterns, nil
}
