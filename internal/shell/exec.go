package shell

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type ExecOption func(*exec.Cmd)

var (
	WithStdout = func(stdout io.Writer) ExecOption {
		return func(c *exec.Cmd) { c.Stdout = stdout }
	}
	WithStderr = func(stderr io.Writer) ExecOption {
		return func(c *exec.Cmd) { c.Stderr = stderr }
	}
	WithStdin = func(stdin io.Reader) ExecOption {
		return func(c *exec.Cmd) { c.Stdin = stdin }
	}
	WithDir = func(dir string) ExecOption {
		return func(c *exec.Cmd) { c.Dir = dir }
	}
)

type Shell struct {
	dir   string
	debug bool
}

func (shell Shell) Debug() Shell {
	shell.debug = true
	return shell
}

func (shell Shell) Dir(dir string) Shell {
	shell.dir = dir
	return shell
}

// Default is the default shell with no set directory.
// Defaulting to running commands in the same directory as the parent Go process.
var Default Shell

func (shell Shell) Execf(ctx context.Context, format string, args []any, opts ...ExecOption) error {
	return shell.Exec(ctx, []string{"sh", "-c", "set -euo pipefail\n\n" + fmt.Sprintf(format, args...)}, opts...)
}

func (shell Shell) Exec(ctx context.Context, args []string, opts ...ExecOption) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	for _, opt := range opts {
		opt(cmd)
	}

	if cmd.Dir == "" {
		cmd.Dir = shell.dir
	}

	var out bytes.Buffer
	cmd.Stdout = makeMultiWriter(cmd.Stdout, &out)
	cmd.Stderr = makeMultiWriter(cmd.Stderr, &out)

	if shell.debug {
		fmt.Println("running:", strings.Join(args, " "))
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: output: %s", err, &out)
	}
	return nil
}

func (shell Shell) ExecCombined(ctx context.Context, args []string, opts ...ExecOption) ([]byte, error) {
	var out bytes.Buffer
	opts = append(opts, WithStderr(&out), WithStdout(&out))
	err := shell.Exec(ctx, args, opts...)
	return bytes.TrimSpace(out.Bytes()), err
}

func (shell Shell) ExecfCombined(ctx context.Context, format string, args []any, opts ...ExecOption) ([]byte, error) {
	var out bytes.Buffer
	opts = append(opts, WithStderr(&out), WithStdout(&out))
	err := shell.Execf(ctx, format, args, opts...)
	return bytes.TrimSpace(out.Bytes()), err
}

func JSONReader(value any) io.Reader {
	data, err := json.Marshal(value)
	if err != nil {
		pr, pw := io.Pipe()
		pw.CloseWithError(err)
		return pr
	}
	return bytes.NewReader(data)
}

func makeMultiWriter(writers ...io.Writer) io.Writer {
	result := make([]io.Writer, 0, len(writers))
	for _, writer := range writers {
		if writer == nil {
			continue
		}
		result = append(result, writer)
	}

	switch len(result) {
	case 0:
		return nil
	case 1:
		return result[0]
	default:
		return io.MultiWriter(result...)
	}
}
