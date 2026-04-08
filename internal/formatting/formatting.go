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

func RenderNames[T Named](writer io.Writer, items []T) error {
	for _, item := range items {
		if _, err := fmt.Fprintln(writer, item.GetName()); err != nil {
			return err
		}
	}
	return nil
}

// AddFormatFlag registers --format / -f for render format (table, json, yaml, names).
func AddFormatFlag(cmd *cobra.Command, format *Format) {
	cmd.Flags().StringVarP((*string)(format), "format", "f", string(FormatTable), "output format, one of: table, json, yaml, names")
}
