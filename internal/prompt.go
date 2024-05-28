package internal

import (
	"slices"

	"github.com/nestoca/survey/v2"
)

func MultiSelect(msg string, options []string) ([]string, error) {
	options = append([]string{}, options...)
	slices.Sort(options)

	ms := &survey.MultiSelect{
		Message: msg,
		Options: options,
	}

	var answer []string
	err := survey.AskOne(ms, &answer)

	return answer, err
}
