package catalog

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Resource interface {
	metadata() Metadata
}

func GetResourceFromNodeFunc(node *yaml.Node) (func(node *yaml.Node) (*Resource, error), error) {
	apiVersion, err := getRequiredMappingStringByName(node, "apiVersion")
	if err != nil {
		return nil, err
	}

	kind, err := getRequiredMappingStringByName(node, "kind")
	if err != nil {
		return nil, err
	}

	switch kind {
	case "Release":
		if apiVersion != ReleaseApiVersion {
			return nil, fmt.Errorf("Release only supports apiVersion %q", ReleaseApiVersion)
		}
		return ReleaseFromNode, nil
	case "Service":
		if apiVersion != ServiceApiVersion {
			return nil, fmt.Errorf("Service only supports apiVersion %q", ServiceApiVersion)
		}
		return ServiceFromNode, nil
	default:
		return nil, fmt.Errorf("kind not supported: %q", kind)
	}
}

func getRequiredMappingStringByName(node *yaml.Node, name string) (string, error) {
	node, err := getRequiredMappingNodeByName(node, name)
	if err != nil {
		return "", err
	}
	return node.Value, nil
}

func getRequiredMappingNodeByName(node *yaml.Node, name string) (*yaml.Node, error) {
	index := findMappingItemByName(node, name)
	if index == -1 {
		return nil, fmt.Errorf("required node %q not found", name)
	}
	return node.Content[index*2+1], nil
}

// findMappingItemByName returns index of  (numbered 0 for first key/value pair,
// 1 for second, and so on) or -1 if not found.
func findMappingItemByName(node *yaml.Node, name string) int {
	for i := 0; i < len(node.Content)/2; i++ {
		if node.Content[i*2].Value == name {
			return i
		}
	}
	return -1
}

func getMetadataName(node *yaml.Node) (string, error) {
	metadata, err := getRequiredMappingNodeByName(node, "metadata")
	if err != nil {
		return "", err
	}
	return getRequiredMappingStringByName(metadata, "name")
}
