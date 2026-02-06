package retry

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func retriable(fn func() error) error {
	var err error
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
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

func RunWithCombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	return Retriable(func() ([]byte, error) {
		var b bytes.Buffer
		cmd.Stdout = &b
		cmd.Stderr = &b
		err := cmd.Run()
		return b.Bytes(), err
	})
}

func Run(cmd *exec.Cmd) error {
	return retriable(func() error {
		return cmd.Run()
	})
}
