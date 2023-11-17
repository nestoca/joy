package xerr

import (
	"strings"
)

type MultiErr struct {
	Errors []error
	Msg    string
	Indent string
}

func (err MultiErr) Unwrap() []error { return err.Errors }

func (err MultiErr) Error() string {
	if err.Msg == "" {
		err.Msg = "error"
	}
	switch len(err.Errors) {
	case 0:
		return err.Msg

	case 1:
		return err.Msg + ": " + err.Errors[0].Error()

	default:
		if err.Indent == "" {
			err.Indent = "  "
		}

		var builder strings.Builder

		builder.WriteString(err.Msg + ":")
		for _, e := range err.Errors {
			builder.WriteString("\n" + indent("- "+e.Error(), err.Indent))
		}

		return builder.String()
	}
}

func indent(value, indent string) string {
	return indent + strings.ReplaceAll(value, "\n", "\n"+indent)
}

func MultiErrWithIndentFrom(msg, indent string, errs ...error) error {
	var nonNilErrs []error
	for _, err := range errs {
		if err != nil {
			nonNilErrs = append(nonNilErrs, err)
		}
	}
	if len(nonNilErrs) == 0 {
		return nil
	}
	return MultiErr{
		Errors: errs,
		Msg:    msg,
		Indent: indent,
	}
}

func MultiErrFrom(msg string, errs ...error) error {
	return MultiErrWithIndentFrom(msg, "  ", errs...)
}
