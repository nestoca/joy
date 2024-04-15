package help

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/nestoca/joy/internal/config"
)

func TestWrapError(t *testing.T) {
	originalStr := "original error"
	originalError := errors.New(originalStr)
	nestedCmdFullName := "sub cmd"
	testCases := []struct {
		name          string
		helps         []config.Help
		originalError error
		expectedError string
	}{
		{
			name:          "nil error",
			originalError: nil,
			expectedError: "",
		},
		{
			name:          "no help",
			helps:         []config.Help{},
			originalError: originalError,
			expectedError: originalStr,
		},
		{
			name: "non-matching command",
			helps: []config.Help{
				{Command: "some command", ErrorPattern: ".*", Message: "some message"},
				{Command: "some other command", ErrorPattern: ".*", Message: "some other message"},
			},
			originalError: originalError,
			expectedError: originalStr,
		},
		{
			name: "non-matching error pattern",
			helps: []config.Help{
				{Command: nestedCmdFullName, ErrorPattern: "some error .* pattern", Message: "some message"},
				{Command: nestedCmdFullName, ErrorPattern: "some other error .* pattern", Message: "some other message"},
			},
			originalError: originalError,
			expectedError: originalStr,
		},
		{
			name: "matching command",
			helps: []config.Help{
				{Command: nestedCmdFullName, Message: "some message"},
				{Message: "default message"},
			},
			originalError: originalError,
			expectedError: originalStr + "\n\nsome message",
		},
		{
			name: "matching error pattern",
			helps: []config.Help{
				{ErrorPattern: "orig.* error", Message: "some message"},
				{Message: "default message"},
			},
			originalError: originalError,
			expectedError: originalStr + "\n\nsome message",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{Helps: tc.helps}
			nestedCmd := getNestedCommand("root", "sub", "cmd")
			nestedCmd.SetContext(config.ToContext(context.TODO(), &cfg))

			actualError := WrapError(nestedCmd, tc.originalError)
			if tc.expectedError == "" {
				require.Nil(t, actualError)
				return
			}
			require.Equal(t, tc.expectedError, actualError.Error())
		})
	}
}

func getNestedCommand(names ...string) *cobra.Command {
	var parent *cobra.Command
	for _, name := range names {
		cmd := &cobra.Command{Use: name}
		if parent != nil {
			parent.AddCommand(cmd)
		}
		parent = cmd
	}
	return parent
}

func TestGetFullCommandName(t *testing.T) {
	testCases := []struct {
		name     string
		cmd      *cobra.Command
		expected string
	}{
		{
			name:     "root command",
			cmd:      getNestedCommand("root"),
			expected: "",
		},
		{
			name:     "nested command",
			cmd:      getNestedCommand("root", "sub"),
			expected: "sub",
		},
		{
			name:     "nested command",
			cmd:      getNestedCommand("root", "sub", "cmd"),
			expected: "sub cmd",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := getFullCommandName(tc.cmd)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestGetHelpMatchingScore(t *testing.T) {
	testCases := []struct {
		name          string
		help          config.Help
		cmdName       string
		errStr        string
		expectedScore int
		expectedError string
	}{
		{
			name:          "default help",
			help:          config.Help{},
			cmdName:       "cmd",
			errStr:        "error",
			expectedScore: 1,
		},
		{
			name:          "help for matching command",
			help:          config.Help{Command: "cmd"},
			cmdName:       "cmd",
			errStr:        "error",
			expectedScore: 6,
		},
		{
			name:          "help for matching error pattern",
			help:          config.Help{ErrorPattern: "err.*"},
			cmdName:       "cmd",
			errStr:        "error",
			expectedScore: 6,
		},
		{
			name:          "help for matching command and error pattern",
			help:          config.Help{Command: "cmd", ErrorPattern: "err.*"},
			cmdName:       "cmd",
			errStr:        "error",
			expectedScore: 11,
		},
		{
			name:          "help for non-matching command, but matching error pattern",
			help:          config.Help{Command: "some specific cmd", ErrorPattern: "err.*"},
			cmdName:       "some other cmd",
			errStr:        "error",
			expectedScore: 0,
		},
		{
			name:          "help for matching command, but non-matching error pattern",
			help:          config.Help{Command: "cmd", ErrorPattern: "specific err.* message"},
			cmdName:       "cmd",
			errStr:        "some other error message",
			expectedScore: 0,
		},
		{
			name:          "help with invalid error pattern",
			help:          config.Help{ErrorPattern: "["},
			expectedError: "error parsing regexp: missing closing ]: `[`",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := getHelpMatchingScore(tc.help, tc.cmdName, tc.errStr)
			if tc.expectedError != "" {
				require.EqualError(t, err, tc.expectedError)
			}
			require.Equal(t, tc.expectedScore, actual)
		})
	}
}
