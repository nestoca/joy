package main

import (
	"regexp"

	"gopkg.in/yaml.v3"
)

var lockMarkerRegex = regexp.MustCompile(`(?i)(?m)^#+\s*lock\s*$`)

func IsCommentLocked(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	return lockMarkerRegex.MatchString(node.HeadComment) || lockMarkerRegex.MatchString(node.LineComment)
}
