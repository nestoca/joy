package promote_test

import (
	"github.com/nestoca/joy/internal/testutils"
	"testing"
)

func TestMain(m *testing.M) {
	testutils.RunTestsInClonedRepo(m, "joy-release-promote-test")
}
