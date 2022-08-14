package pattern

import (
	"fmt"
	"regexp"
)

// Pattern allows to specify one or multiple resources in a given order, potentially using wildcards.
// See: https://nestoca.atlassian.net/wiki/spaces/DEVOPS/pages/2270691380/Design+Joy+CLI#Name-patterns
type Pattern struct {
	regex regexp.Regexp
}

type Patterns []Pattern

func NewFromString(str string) (*Pattern, error) {
	// 1. Split into token at commas
	// 2. For each token:
	//  2a. If starts with '!' remove it and set negation flag
	//  2b. Replace '*' with '.*'
	//  2c. Surround with '^...$'
	//  2d. If negation flag set, surround with '(?!(...))' (see: https://newbedev.com/how-to-negate-the-whole-regex)
	//  2e. Parse as regex
	return nil, fmt.Errorf("not implemented")
}

func (p Pattern) Matches(str string) bool {
	panic(fmt.Errorf("not implemented"))
}

func (p Patterns) Match(str string) bool {
	panic(fmt.Errorf("not implemented"))
}
