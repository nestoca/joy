package yml

import (
	"regexp"

	"gopkg.in/yaml.v3"
)

var lockMarkerRegex = regexp.MustCompile(`(?i)(?m)^#+\s*lock\s*$`)

func IsLocked(key, value *yaml.Node) bool {
	commentLocked := IsCommentLocked(key) || IsCommentLocked(value)
	tagLocked := value != nil && value.Tag == "!lock"
	return commentLocked || tagLocked
}

func IsCommentLocked(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	return lockMarkerRegex.MatchString(node.HeadComment) || lockMarkerRegex.MatchString(node.LineComment)
}
