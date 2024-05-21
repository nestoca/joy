package help

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/style"
)

const allCommandsKey = "~all"

func WrapError(cmd *cobra.Command, cmdError error) error {
	if cmdError == nil {
		return nil
	}

	message, err := getMessageFromCommand(cmd, cmdError)
	if err != nil {
		return err
	}
	if message == "" {
		return cmdError
	}

	return fmt.Errorf("%w\n\n%s", cmdError, message)
}

func AugmentCommandHelp(cmd *cobra.Command, io internal.IO) {
	baseHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		baseHelpFunc(cmd, args)

		message, err := getMessageFromCommand(cmd, nil)
		if err != nil {
			_, _ = fmt.Fprintln(io.Err, err)
		}
		if message != "" {
			_, _ = fmt.Fprintln(io.Out, "\n"+style.Notice(message))
		}
	})
}

func getMessageFromCommand(cmd *cobra.Command, cmdError error) (string, error) {
	cfg := config.FromContext(cmd.Context())
	fullCommandName := getFullCommandName(cmd)

	errStr := ""
	if cmdError != nil {
		errStr = cmdError.Error()
	}

	return getMessage(cfg.Helps, fullCommandName, errStr)
}

func getMessage(helps map[string][]config.Help, fullCommandName string, errStr string) (string, error) {
	var messages []string
	cmdAndAllHelps := append(helps[fullCommandName], helps[allCommandsKey]...)
	for _, help := range cmdAndAllHelps {
		if help.ErrorPattern != "" {
			isMatched, err := regexp.MatchString(help.ErrorPattern, errStr)
			if err != nil {
				return "", fmt.Errorf("matching error pattern %q: %w", help.ErrorPattern, err)
			}
			if !isMatched {
				continue
			}
		}
		messages = append(messages, strings.TrimSpace(help.Message))
	}

	return strings.Join(messages, "\n\n"), nil
}

func getFullCommandName(cmd *cobra.Command) string {
	_, subPath, _ := strings.Cut(cmd.CommandPath(), " ")
	return subPath
}
