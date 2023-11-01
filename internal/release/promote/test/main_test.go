package promote_test

import (
	"testing"

	"github.com/nestoca/joy/internal/testutils"
)

func TestMain(m *testing.M) {
	testutils.RunTestsInClonedRepo(m, "joy-release-promote-test")
}
