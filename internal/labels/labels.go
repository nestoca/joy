// Package labels matches Kubernetes-style metadata labels via `key` / `key=value` selectors.
package labels

import (
	"fmt"
	"strings"
)

// Selector matches a metadata label by key, and optionally by exact value.
type Selector struct {
	key      string
	value    string
	hasValue bool
}

func (s Selector) String() string {
	if s.hasValue {
		return s.key + "=" + s.value
	}
	return s.key
}

// matches reports whether labelSet contains this selector's key (and, when a value is
// specified, that it equals that value).
func (s Selector) matches(labelSet map[string]string) bool {
	value, ok := labelSet[s.key]
	if !ok {
		return false
	}
	return !s.hasValue || value == s.value
}

// ParseSelectors parses `key` / `key=value` specs into selectors. A bare `key` matches the
// label regardless of its value; `key=value` requires an exact match. An empty key is rejected.
func ParseSelectors(specs []string) ([]Selector, error) {
	selectors := make([]Selector, 0, len(specs))
	for _, spec := range specs {
		key, value, hasValue := strings.Cut(spec, "=")
		// Trim surrounding whitespace so a padded key (e.g. " nesto.ca/preview") still
		// matches the actual metadata key rather than silently never matching.
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("invalid label selector %q: key is empty", spec)
		}
		selectors = append(selectors, Selector{key: key, value: value, hasValue: hasValue})
	}
	return selectors, nil
}

// FirstMatch returns the first selector matching the given label set, if any.
func FirstMatch(selectors []Selector, labelSet map[string]string) (Selector, bool) {
	for _, selector := range selectors {
		if selector.matches(labelSet) {
			return selector, true
		}
	}
	return Selector{}, false
}
