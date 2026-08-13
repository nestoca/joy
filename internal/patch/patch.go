// Package patch applies RFC 6902 JSON Patch operations to a YAML release tree.
//
// The patch itself is applied by a standard JSON Patch library, so serialization to JSON is
// required — which drops YAML custom tags (e.g. !lock), comments and key order. Those are
// restored from the original tree afterwards via yml.CopyMetadata. It also exposes a few
// node-level setters used for built-in transforms that must run before the JSON round-trip.
package patch

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"gopkg.in/yaml.v3"
	sigsyaml "sigs.k8s.io/yaml"
)

// Op is a single RFC 6902 operation. Value is an arbitrary JSON value (set for add/replace);
// From is only set for move/copy.
type Op struct {
	Op    string `json:"op" yaml:"op"`
	Path  string `json:"path" yaml:"patch"`
	From  string `json:"from,omitempty" yaml:"from,omitempty"`
	Value any    `json:"value,omitempty" yaml:"value,omitempty"`
}

// Apply applies the ops to doc (a document node) and returns the resulting document. The patch is
// applied via a JSON Patch library; custom tags, comments and key order are then restored from
// doc, and styles are cleared so the result serializes as clean block YAML. When there are no
// ops, doc is returned unchanged.
func Apply[T any](doc *T, ops []Op) (*T, error) {
	if len(ops) == 0 {
		return doc, nil
	}

	docJSON, err := toJSON(doc)
	if err != nil {
		return nil, fmt.Errorf("converting release to json: %w", err)
	}

	patchJSON, err := json.Marshal(ops)
	if err != nil {
		return nil, fmt.Errorf("encoding patch: %w", err)
	}

	patch, err := jsonpatch.DecodePatch(patchJSON)
	if err != nil {
		return nil, fmt.Errorf("decoding patch: %w", err)
	}

	patchedJSON, err := patch.Apply(docJSON)
	if err != nil {
		return nil, fmt.Errorf("applying patch: %w", err)
	}

	var patched T
	if err := yaml.Unmarshal(patchedJSON, &patched); err != nil {
		return nil, fmt.Errorf("parsing patched result: %w", err)
	}

	return &patched, nil
}

func toJSON(value any) ([]byte, error) {
	data, err := yaml.Marshal(value)
	if err != nil {
		return nil, err
	}
	return sigsyaml.YAMLToJSON(data)
}
