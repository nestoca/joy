package yml

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

func TreeToYaml(tree *yaml.Node, indent int) (string, error) {
	buf := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buf)
	encoder.SetIndent(indent)
	err := encoder.Encode(tree)
	if err != nil {
		return "", fmt.Errorf("encoding node tree to yaml: %w", err)
	}
	return buf.String(), nil
}
