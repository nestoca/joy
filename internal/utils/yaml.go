package utils

import (
	"bytes"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"reflect"
	"strings"
)

// FindNode traverses a given yaml.Node to locate the value Node of the provided path. The path is interpreted as being
// relative to the given yaml.Node.
func FindNode(node *yaml.Node, path string) (resultNode *yaml.Node, err error) {

	var findNodeErr error

	// If node is a DocumentNode, we need to retrieve the MappingNode from its Content and traverse it.
	if node.Kind == yaml.DocumentNode && len(node.Content) == 1 {
		resultNode, findNodeErr = findNode(node.Content[0], segmentPath(path))
	} else {
		resultNode, findNodeErr = findNode(node, segmentPath(path))
	}

	if findNodeErr != nil {
		return nil, fmt.Errorf("node not found for path '%s': %w", path, findNodeErr)
	}

	return resultNode, nil
}

func findNode(node *yaml.Node, pathSegments []string) (*yaml.Node, error) {
	// Condition is i+1 < len(node.Content) as there always need to be at least 2 entries left in the node.Content for
	// traversal to work, because each key and its associated value are stored in two consecutive nodes in the slice.
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		// If the value of the key node matches the first key in the pathSegments, its valueNode contains what we're
		// looking for (Either the value itself or the next child to search)
		if keyNode.Value == pathSegments[0] {
			// If there is only 1 segment left in pathSegments, then this is the droid we are looking for.
			if len(pathSegments) == 1 {
				return valueNode, nil
			}

			// MappingNodes contain more key/value pairings, so we'll recurse into the next level of the path to search.
			if valueNode.Kind == yaml.MappingNode {
				return findNode(valueNode, pathSegments[1:])
			}
		}
	}

	return nil, fmt.Errorf("key '%s' does not exist", pathSegments[0])
}

func segmentPath(path string) []string {
	segments := strings.Split(path, ".")
	if len(segments) > 0 && segments[0] == "" {
		segments = segments[1:]
	}

	return segments
}

func EncodeYaml(obj any) ([]byte, error) {
	if reflect.ValueOf(obj).Kind() != reflect.Ptr {
		return nil, errors.New("obj must be a pointer")
	}

	var b bytes.Buffer

	yamlEncoder := yaml.NewEncoder(&b)
	yamlEncoder.SetIndent(2)
	err := yamlEncoder.Encode(obj)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
