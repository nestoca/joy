package retry

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetriableReturnsAbortedErrorWithoutRetrying(t *testing.T) {
	errNotFound := errors.New("not found")

	var calls int
	_, err := Retriable(func() (string, error) {
		calls++
		return "", fmt.Errorf("cloning: %w", Abort(errNotFound))
	})

	require.ErrorIs(t, err, errNotFound)
	require.Equal(t, 1, calls)
}
