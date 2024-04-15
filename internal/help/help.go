package help

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal/config"
)

func WrapError(cmd *cobra.Command, cmdError error) error {
	if cmdError == nil {
		return nil
	}

	message, err := getMessage(cmd, cmdError)
	if err != nil {
		return err
	}
	if message == "" {
		return cmdError
	}

	return fmt.Errorf("%w\n\n%s", cmdError, message)
}

func AugmentCommandHelp(cmd *cobra.Command) {
	baseHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		baseHelpFunc(cmd, args)

		message, err := getMessage(cmd, nil)
		if err != nil {
			panic(err)
		}
		if message != "" {
			fmt.Println("\n" + message)
		}
	})
}

func getMessage(cmd *cobra.Command, cmdError error) (string, error) {
	cfg := config.FromContext(cmd.Context())
	fullCommandName := getFullCommandName(cmd)
	errStr := ""
	if cmdError != nil {
		errStr = cmdError.Error()
	}
	bestScore := 0
	bestMessage := ""
	for _, h := range cfg.Helps {
		score, err := getHelpMatchingScore(h, fullCommandName, errStr)
		if err != nil {
			return "", err
		}
		if score > bestScore {
			bestScore = score
			bestMessage = strings.TrimSpace(h.Message)
		}
	}
	return bestMessage, nil
}

func getHelpMatchingScore(h config.Help, fullCommandName, errStr string) (int, error) {
	score := 1
	if h.Command != "" {
		if h.Command != fullCommandName {
			return 0, nil
		}
		score += 5
	}
	if h.ErrorPattern != "" {
		isMatched, err := regexp.MatchString(h.ErrorPattern, errStr)
		if err != nil {
			return 0, err
		}
		if !isMatched {
			return 0, nil
		}
		score += 5
	}
	return score, nil
}

func getFullCommandName(cmd *cobra.Command) string {
	if !cmd.HasParent() {
		return ""
	}
	if !cmd.Parent().HasParent() {
		return cmd.Name()
	}
	return getFullCommandName(cmd.Parent()) + " " + cmd.Name()
}
