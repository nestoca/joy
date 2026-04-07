package formatting

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJson  Format = "json"
	FormatYaml  Format = "yaml"
	FormatNames Format = "names"
)

func (f *Format) String() string {
	if f == nil {
		return ""
	}
	return string(*f)
}

func (f *Format) Set(value string) error {
	format := Format(value)
	switch format {
	case FormatTable, FormatJson, FormatYaml, FormatNames:
		*f = format
		return nil
	default:
		return fmt.Errorf("invalid format %q (must be one of: table, json, yaml, names)", value)
	}
}

func (f *Format) Type() string {
	return "format"
}

func RenderJson(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func RenderYaml(writer io.Writer, value any) error {
	encoder := yaml.NewEncoder(writer)
	encoder.SetIndent(2)
	return encoder.Encode(value)
}

func RenderNames[T NamedObject](writer io.Writer, items []T) error {
	for _, item := range items {
		if _, err := fmt.Fprintln(writer, item.GetName()); err != nil {
			return err
		}
	}
	return nil
}

// AddFormatFlag registers --format / -f for render format (table, json, yaml, names).
func AddFormatFlag(cmd *cobra.Command, format *Format) {
	cmd.Flags().VarP(format, "format", "f", "format (table, json, yaml, names; defaults to table)")
	_ = cmd.Flags().Set("format", string(FormatTable))
}
