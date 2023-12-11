package catalog

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/internal/yml"
)

func validateTagsForFiles(files []*yml.File) error {
	var errs []error
	for _, file := range files {
		if err := validateTags(file.Tree); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", file.Path, err))
		}
	}
	return errors.Join(errs...)
}

func validateTags(node *yaml.Node) error {
	if tags := buildUnknownTagsList(node); len(tags) > 0 {
		sort.Strings(tags)
		return fmt.Errorf("unknown tag(s): %s", strings.Join(tags, ", "))
	}
	return nil
}

func buildUnknownTagsList(node *yaml.Node) []string {
	var (
		set  = buildUnknownTagsSet(node, nil)
		list = make([]string, 0, len(set))
	)
	for value := range set {
		list = append(list, value)
	}
	return list
}

func buildUnknownTagsSet(node *yaml.Node, set map[string]struct{}) map[string]struct{} {
	if set == nil {
		set = make(map[string]struct{})
	}

	if !slices.Contains(yml.KnownTags, node.Tag) {
		set[node.Tag] = struct{}{}
	}

	for _, content := range node.Content {
		set = buildUnknownTagsSet(content, set)
	}

	return set
}
