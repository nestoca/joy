package releasing

import "strings"

type Filter interface {
	Match(rel *Release) bool
}

type NamePatternFilter struct {
	ReleaseNames []string
}

func NewNamePatternFilter(pattern string) *NamePatternFilter {
	return &NamePatternFilter{
		ReleaseNames: strings.Split(pattern, ","),
	}
}

func (f *NamePatternFilter) Match(rel *Release) bool {
	for _, name := range f.ReleaseNames {
		if rel.Metadata.Name == name {
			return true
		}
	}
	return false
}

type SpecificReleasesFilter struct {
	ReleaseNames []string
}

func NewSpecificReleasesFilter(releaseNames []string) *SpecificReleasesFilter {
	return &SpecificReleasesFilter{
		ReleaseNames: releaseNames,
	}
}

func (f *SpecificReleasesFilter) Match(rel *Release) bool {
	for _, name := range f.ReleaseNames {
		if rel.Name == name {
			return true
		}
	}
	return false
}
