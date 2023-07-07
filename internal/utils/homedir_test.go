package utils_test

import (
	"fmt"
	"github.com/nestoca/joy/internal/utils"
	"github.com/stretchr/testify/assert"
	"os/user"
	"path/filepath"
	"testing"
)

func TestResolvePathWithoutTilde(t *testing.T) {
	usr, err := user.Current()
	assert.Nil(t, err)
	assert.NotNil(t, usr)

	path := "/foo/bar"

	resolvedPath, err := utils.ResolvePath(path)
	assert.Equal(t, path, resolvedPath)
	assert.Nil(t, err)
}

func TestResolvePathWithJustTilde(t *testing.T) {
	usr, err := user.Current()
	assert.Nil(t, err)
	assert.NotNil(t, usr)

	resolvedPath, err := utils.ResolvePath("~")
	assert.Equal(t, usr.HomeDir, resolvedPath)
	assert.Nil(t, err)
}

func TestResolvePathWithTildePrefix(t *testing.T) {
	usr, err := user.Current()
	assert.Nil(t, err)
	assert.NotNil(t, usr)

	pathInHomeDir := "foo/bar"

	resolvedPath, err := utils.ResolvePath(fmt.Sprintf("~/%s", pathInHomeDir))
	assert.Equal(t, filepath.Join(usr.HomeDir, pathInHomeDir), resolvedPath)
	assert.Nil(t, err)
}
