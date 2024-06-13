package internal

import (
	"encoding/json"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/format"

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

func ValidateAgainstSchema(schema cue.Value, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	var generic any
	if err := json.Unmarshal(data, &generic); err != nil {
		return err
	}

	baseValue := schema.Context().Encode(generic)
	if err := schema.Unify(baseValue).Validate(cue.Final(), cue.Concrete(true)); err != nil {
		var errs []error
		for _, e := range cueerrors.Errors(err) {
			errs = append(errs, e)
		}
		return xerr.MultiErrFrom("", errs...)
	}

	return nil
}
