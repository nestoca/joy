package testutils

import (
	"os"
	"strconv"
	"testing"
)

var ci = func() bool {
	isCI, _ := strconv.ParseBool(os.Getenv("CI"))
	return isCI
}()

func SkipIfCI(t *testing.T) {
	if ci {
		t.Skipf("skipping test %s: test has been flagged not to run in CI", t.Name())
	}
}
