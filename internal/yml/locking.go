package yml

import (
	"regexp"

	"gopkg.in/yaml.v3"
)

var lockMarkerRegex = regexp.MustCompile(`(?i)(?m)^#+\s*lock\s*$`)

func IsKeyValueLocked(key, value *yaml.Node) bool {
	return IsLocked(key) || IsLocked(value)
}

func IsLocked(node *yaml.Node) bool {
	return IsTagLocked(node) || IsCommentLocked(node)
}

func IsTagLocked(node *yaml.Node) bool {
	return node != nil && node.Tag == "!lock"
}

func IsCommentLocked(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	return lockMarkerRegex.MatchString(node.HeadComment) || lockMarkerRegex.MatchString(node.LineComment)
}
