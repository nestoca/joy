package internal

import (
	"io"

	"github.com/spf13/cobra"
)

type IO struct {
	Out io.Writer
	Err io.Writer
	In  io.Reader
}

func IoFromCommand(cmd *cobra.Command) IO {
	return IO{
		Out: cmd.OutOrStdout(),
		Err: cmd.ErrOrStderr(),
		In:  cmd.InOrStdin(),
	}
}
