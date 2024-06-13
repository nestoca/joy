package v1alpha1

import (
	_ "embed"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	"github.com/davidmdm/x/xerr"

	"github.com/nestoca/joy/internal"
)

//go:embed schemas.cue
var schemaText string

type Schemas struct {
	Release     cue.Value
	Environment cue.Value
	Project     cue.Value
}

var schemas Schemas

func init() {
	runtime := cuecontext.New()
	schema := runtime.CompileString(schemaText)

	var errs []error
	for key, ptr := range map[string]*cue.Value{
		"environment": &schemas.Environment,
		"project":     &schemas.Project,
		"release":     &schemas.Release,
	} {
		*ptr = schema.LookupPath(cue.MakePath(cue.Def(key)))
		if err := ptr.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", key, err))
		}
	}

	if err := xerr.MultiErrOrderedFrom("validating v1alpha1 schemas", errs...); err != nil {
		panic(err)
	}
}

func ReleaseSpecification() string     { return internal.StringifySchema(schemas.Release) }
func EnvironmentSpecification() string { return internal.StringifySchema(schemas.Environment) }
func ProjectSpecification() string     { return internal.StringifySchema(schemas.Project) }
