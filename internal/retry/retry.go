package retry

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type abortError struct {
	err error
}

func (e *abortError) Error() string {
	return e.err.Error()
}

func (e *abortError) Unwrap() error {
	return e.err
}

// Abort wraps given error so that it is returned right away instead of being retried, including by outer retries of
// errors wrapping it.
func Abort(err error) error {
	return &abortError{err: err}
}

func retriable(fn func() error) error {
	var err error
	maxRetries := 5
	for i := range maxRetries {
		err = fn()
		if err == nil {
			return nil
		}
		var abort *abortError
		if errors.As(err, &abort) {
			return err
		}
		if i < maxRetries-1 {
			secs := i + 2
			_, _ = fmt.Fprintf(os.Stderr, "Retrying in %d seconds. error: %v\n", secs, err)
			time.Sleep(time.Second * time.Duration(secs))
		}
	}
	return err
}

func Retriable[T any](fn func() (T, error)) (T, error) {
	var result T
	err := retriable(func() error {
		var err error
		result, err = fn()
		return err
	})
	return result, err
}

func RunWithCombinedOutput(cmd func() *exec.Cmd) ([]byte, error) {
	return Retriable(func() ([]byte, error) { return cmd().CombinedOutput() })
}

func Run(cmd func() *exec.Cmd) error {
	return retriable(func() error { return cmd().Run() })
}
