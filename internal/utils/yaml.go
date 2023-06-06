package utils

import (
	"fmt"
	"strings"
)

// TraverseYAML Traverse the YAML data using the provided path and return the value
func TraverseYAML(data interface{}, path string) (interface{}, error) {
	segments := segmentPath(path)

	current := data
	for _, key := range segments {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[key]
		} else {
			return nil, fmt.Errorf("invalid path segment: %s", key)
		}
	}
	return current, nil
}

// SetYAMLValue Set the value at the provided path in the YAML data
func SetYAMLValue(data interface{}, path string, value interface{}) error {
	segments := segmentPath(path)

	current := data
	for i, key := range segments {
		if i == len(segments)-1 {
			if m, ok := current.(map[string]interface{}); ok {
				m[key] = value
				return nil
			}
			return fmt.Errorf("invalid path segment: %s", key)
		}
		if m, ok := current.(map[string]interface{}); ok {
			current = m[key]
		} else {
			return fmt.Errorf("invalid path segment: %s", key)
		}
	}
	return nil
}

func segmentPath(path string) []string {
	segments := strings.Split(path, ".")
	if len(segments) > 0 && segments[0] == "" {
		segments = segments[1:]
	}

	return segments
}
