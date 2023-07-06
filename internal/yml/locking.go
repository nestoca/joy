package yml

import (
	"gopkg.in/yaml.v3"
	"regexp"
)

var lockMarkerRegex = regexp.MustCompile(`(?m)^##\s*lock\s*$`)

func IsLocked(keyNode, valueNode *yaml.Node) bool {
	isKeyNodeMarkedAsLocked :=
		keyNode != nil && (lockMarkerRegex.MatchString(keyNode.HeadComment) ||
			lockMarkerRegex.MatchString(keyNode.LineComment))
	isValueNodeMarkedAsLocked :=
		valueNode != nil &&
			(valueNode.Kind == yaml.ScalarNode && lockMarkerRegex.MatchString(valueNode.LineComment))
	return isKeyNodeMarkedAsLocked || isValueNodeMarkedAsLocked
}
