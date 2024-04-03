package testutils

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sergi/go-diff/diffmatchpatch"
)

type DiffType int

const (
	Missing DiffType = iota
	Unexpected
	Different
)

type FileDiff struct {
	Path string
	Type DiffType
	Diff []diffmatchpatch.Diff
}

func (fd FileDiff) String() string {
	if fd.Type == Missing {
		return fmt.Sprintf("Missing: %s", fd.Path)
	}
	if fd.Type == Unexpected {
		return fmt.Sprintf("Unexpected: %s", fd.Path)
	}
	dmp := diffmatchpatch.New()
	return fmt.Sprintf("Differences in %s:\n———\n%s\n———", fd.Path, dmp.DiffPrettyText(fd.Diff))
}

type FileDiffs []FileDiff

func (fds FileDiffs) String() string {
	var buffer bytes.Buffer
	for i, fd := range fds {
		if i > 0 {
			buffer.WriteString("\n")
		}
		buffer.WriteString(fd.String())
	}
	return buffer.String()
}

func CompareDirectories(expectedDir, actualDir string) (FileDiffs, error) {
	var diffs []FileDiff

	dmp := diffmatchpatch.New()

	err := filepath.Walk(expectedDir, func(expectedPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(expectedDir, expectedPath)
		if err != nil {
			return err
		}
		actualPath := filepath.Join(actualDir, relPath)

		actualInfo, err := os.Stat(actualPath)
		if os.IsNotExist(err) {
			diffs = append(diffs, FileDiff{Path: relPath, Type: Missing})
			return nil
		} else if err != nil {
			return err
		}

		if info.IsDir() || actualInfo.IsDir() {
			return nil
		}

		expectedContent, err := os.ReadFile(expectedPath)
		if err != nil {
			return err
		}
		actualContent, err := os.ReadFile(actualPath)
		if err != nil {
			return err
		}

		diff := dmp.DiffMain(string(expectedContent), string(actualContent), false)
		if len(diff) > 1 {
			diffs = append(diffs, FileDiff{Path: relPath, Type: Different, Diff: diff})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(actualDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(actualDir, path)
		if err != nil {
			return err
		}
		expectedPath := filepath.Join(expectedDir, relPath)

		_, err = os.Stat(expectedPath)
		if os.IsNotExist(err) {
			diffs = append(diffs, FileDiff{Path: relPath, Type: Unexpected})
			return nil
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return diffs, nil
}
