package internal

import (
	"fmt"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/format"
	"gopkg.in/yaml.v3"

	"github.com/davidmdm/x/xerr"
)

func StringifySchema(value cue.Value) string {
	node := value.Syntax(
		cue.Concrete(false), // allow incomplete values
		cue.Definitions(false),
		cue.Hidden(true),
		cue.Optional(true),
		cue.Attributes(false),
		cue.Docs(true),
	)

	out, _ := format.Node(node, format.TabIndent(false), format.UseSpaces(2))
	return string(out)
}

func ValidateAgainstSchema(schema cue.Value, node *yaml.Node) error {
	var value any
	if err := node.Decode(&value); err != nil {
		return err
	}

	value = JsonCompat(value)

	baseValue := schema.Context().Encode(value)
	if err := schema.Unify(baseValue).Validate(cue.Final(), cue.Concrete(true)); err != nil {
		var errs []error
		for _, e := range cueerrors.Errors(err) {
			errs = append(errs, e)
		}
		return xerr.MultiErrFrom("", errs...)
	}

	return nil
}

// JsonCompat takes a generic as returned from yaml.Unmarshal and returns a new instance with all map[any]any converted to map[string]any
// for compatiblilty with Apis that only support JSON generic objects. (Objects marshalled from json only support string keys).
func JsonCompat(value any) any {
	switch value := value.(type) {
	case []any:
		copy := make([]any, len(value))
		for i, elem := range value {
			copy[i] = JsonCompat(elem)
		}
		return copy
	case map[string]any:
		copy := make(map[string]any, len(value))
		for k, v := range value {
			copy[k] = JsonCompat(v)
		}
		return copy
	case map[any]any:
		copy := make(map[string]any, len(value))
		for k, v := range value {
			copy[fmt.Sprint(k)] = JsonCompat(v)
		}
		return copy
	default:
		return value
	}
}
