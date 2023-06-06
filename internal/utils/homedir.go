package utils

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strings"
)

func ResolvePath(path string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("could not resolve path: %w", err)
	}

	dir := usr.HomeDir
	if path == "~" {
		return dir, nil
	} else if strings.HasPrefix(path, "~/") {
		return filepath.Join(dir, path[2:]), nil
	}

	return path, nil
}
