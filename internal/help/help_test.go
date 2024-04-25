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
		helps         map[string][]config.Help
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
			helps:         map[string][]config.Help{},
			originalError: originalError,
			expectedError: originalStr,
		},
		{
			name: "non-matching command",
			helps: map[string][]config.Help{
				"some command": {
					{ErrorPattern: ".*", Message: "some message"},
				},
				"some other command": {
					{ErrorPattern: ".*", Message: "some other message"},
				},
			},
			originalError: originalError,
			expectedError: originalStr,
		},
		{
			name: "non-matching error pattern",
			helps: map[string][]config.Help{
				nestedCmdFullName: {
					{ErrorPattern: "some error .* pattern", Message: "some message"},
					{ErrorPattern: "some other error .* pattern", Message: "some other message"},
				},
			},
			originalError: originalError,
			expectedError: originalStr,
		},
		{
			name: "matching command",
			helps: map[string][]config.Help{
				allCommandsKey: {
					{Message: "all commands message"},
				},
				nestedCmdFullName: {
					{Message: "some message"},
				},
			},
			originalError: originalError,
			expectedError: originalStr + "\n\nsome message\n\nall commands message",
		},
		{
			name: "matching error pattern",
			helps: map[string][]config.Help{
				allCommandsKey: {
					{Message: "all commands message"},
					{ErrorPattern: "orig.* error", Message: "some message"},
				},
			},
			originalError: originalError,
			expectedError: originalStr + "\n\nall commands message\n\nsome message",
		},
		{
			name: "non-matching error pattern",
			helps: map[string][]config.Help{
				allCommandsKey: {
					{Message: "all commands message"},
					{ErrorPattern: "some error .* pattern", Message: "some message"},
				},
			},
			originalError: originalError,
			expectedError: originalStr + "\n\nall commands message",
		},
		{
			name: "trimming leading and trailing whitespace",
			helps: map[string][]config.Help{
				allCommandsKey: {
					{Message: "\n\n\tall commands message\n\n\t"},
				},
				nestedCmdFullName: {
					{Message: "\nsome message\n"},
				},
			},
			originalError: originalError,
			expectedError: originalStr + "\n\nsome message\n\nall commands message",
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

func TestGetHelpMessage(t *testing.T) {
	testCases := []struct {
		name            string
		help            map[string][]config.Help
		cmdName         string
		errStr          string
		expectedMessage string
		expectedError   string
	}{
		{
			name:            "default help",
			help:            map[string][]config.Help{},
			cmdName:         "cmd",
			errStr:          "error",
			expectedMessage: "",
		},
		{
			name:            "help for matching command",
			help:            map[string][]config.Help{"cmd": {{Message: "message"}}},
			cmdName:         "cmd",
			errStr:          "error",
			expectedMessage: "message",
		},
		{
			name:            "help for matching error pattern",
			help:            map[string][]config.Help{allCommandsKey: {{ErrorPattern: "err.*", Message: "message"}}},
			cmdName:         "cmd",
			errStr:          "error",
			expectedMessage: "message",
		},
		{
			name:            "help for matching command and error pattern",
			help:            map[string][]config.Help{"cmd": {{ErrorPattern: "err.*", Message: "message"}}},
			cmdName:         "cmd",
			errStr:          "error",
			expectedMessage: "message",
		},
		{
			name:            "help for non-matching command, but matching error pattern",
			help:            map[string][]config.Help{"some specific cmd": {{ErrorPattern: "err.*", Message: "message"}}},
			cmdName:         "some other cmd",
			errStr:          "error",
			expectedMessage: "",
		},
		{
			name:            "help for matching command, but non-matching error pattern",
			help:            map[string][]config.Help{"cmd": {{ErrorPattern: "specific err.* pattern", Message: "message"}}},
			cmdName:         "cmd",
			errStr:          "some other error message",
			expectedMessage: "",
		},
		{
			name: "help for all commands and matching command and error pattern",
			help: map[string][]config.Help{
				allCommandsKey: {
					{Message: "all commands message"},
					{ErrorPattern: "err.*", Message: "all commands error message"},
				},
				"cmd": {
					{Message: "message"},
					{ErrorPattern: "err.*", Message: "error message"},
				},
			},
			cmdName:         "cmd",
			errStr:          "error",
			expectedMessage: "message\n\nerror message\n\nall commands message\n\nall commands error message",
		},
		{
			name:          "help with invalid error pattern",
			help:          map[string][]config.Help{"cmd": {{ErrorPattern: "["}}},
			cmdName:       "cmd",
			expectedError: "matching error pattern \"[\": error parsing regexp: missing closing ]: `[`",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := getMessage(tc.help, tc.cmdName, tc.errStr)
			if tc.expectedError != "" {
				require.EqualError(t, err, tc.expectedError)
			}
			require.Equal(t, tc.expectedMessage, actual)
		})
	}
}
