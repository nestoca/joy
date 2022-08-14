package catalog

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

const ReleaseApiVersion = "joy.nesto.ca/v2"

type Release struct {
	Metadata Metadata
	Spec     ReleaseSpec
}

type Chart struct {
	Repo    string
	Version string
}

type Image struct {
	Name string
	Tag  string
}

type ReleaseSpec struct {
	ServiceName string                 `yaml:"service"`
	Service     Service                `yaml:""`
	Chart       Chart                  `yaml:"chart"`
	Image       Image                  `yaml:"image"`
	Installed   bool                   `yaml:"installed"`
	Values      map[string]interface{} `yaml:"values"`
}

func ReleaseFromNode(node *yaml.Node) (*Resource, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *Release) ToNode() (*yaml.Node, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *Release) metadata() Metadata {
	return r.Metadata
}
