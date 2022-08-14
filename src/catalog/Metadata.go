package catalog

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Metadata struct {
	Name        string            `yaml:"name"`
	Annotations map[string]string `yaml:"annotations"`
}

func (m Metadata) display() string {
	display, ok := m.Annotations["display"]
	if ok {
		return display
	}
	return m.Name
}

func (m Metadata) description() string {
	description, ok := m.Annotations["description"]
	if ok {
		return description
	}
	return ""
}

func (m Metadata) tags() ([]string, error) {
	tagsStr, ok := m.Annotations["tags"]
	if ok {
		var tags []string
		err := yaml.Unmarshal([]byte(tagsStr), &tags)
		if err != nil {
			return nil, fmt.Errorf("tags annotation must be an array of strings: %w", err)
		}
		return tags, nil
	}
	// No tags defined in annotations, but that's OK
	return nil, nil
}
