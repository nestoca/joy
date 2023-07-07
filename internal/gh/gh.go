package gh

import (
	"github.com/cli/cli/pkg/cmd/pr/create"
	"github.com/cli/cli/pkg/cmdutil"
	"github.com/cli/cli/pkg/iostreams"
	"os"
)

// CreatePullRequest creates a pull request.
func CreatePullRequest(args ...string) error {
	f := &cmdutil.Factory{
		IOStreams: iostreams.System(),
	}

	cmd := create.NewCmdCreate(f, func(o *create.CreateOptions) error {
		return nil
	})

	cmd.SetArgs(args)
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	cmd.SetIn(os.Stdin)
	err := cmd.Execute()
	return err
}
