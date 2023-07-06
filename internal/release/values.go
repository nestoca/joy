package release

import "gopkg.in/yaml.v3"

type ValuesFile struct {
	// FilePath is the path to the values file.
	FilePath string `yaml:"-"`

	// Yaml is the raw yaml of the values file.
	Yaml string `yaml:"-"`

	// Node is the yaml root node of the values file.
	Node *yaml.Node `yaml:"-"`

	// Hash is the hash of the values file.
	Hash uint64 `yaml:"-"`
}
