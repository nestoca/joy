package catalog

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

const ServiceApiVersion = "joy.nesto.ca/v1"

type Service struct {
	Metadata Metadata
	Spec     ServiceSpec
}

type ServiceSpec struct {
	// Type determines which template to use for creating the first Release of this service (eg: "go-service")
	// It corresponds to the name of the jen template used when creating service.
	Type string
}

func ServiceFromNode(node *yaml.Node) (*Resource, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *Service) ToNode() (*yaml.Node, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *Service) metadata() Metadata {
	return s.Metadata
}
