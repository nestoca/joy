package retry

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func retriable(fn func() error) error {
	var err error
	maxRetries := 5
	for i := range maxRetries {
		err = fn()
		if err == nil {
			return nil
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
