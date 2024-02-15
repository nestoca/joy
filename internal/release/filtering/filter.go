package filtering

import (
	"strings"

	"github.com/nestoca/joy/api/v1alpha1"
)

type Filter interface {
	Match(rel *v1alpha1.Release) bool
}

type NamePatternFilter struct {
	ReleaseNames []string
}

func NewNamePatternFilter(pattern string) *NamePatternFilter {
	return &NamePatternFilter{
		ReleaseNames: strings.Split(pattern, ","),
	}
}

func (f *NamePatternFilter) Match(rel *v1alpha1.Release) bool {
	for _, name := range f.ReleaseNames {
		if rel.Name == name {
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

func (f *SpecificReleasesFilter) Match(rel *v1alpha1.Release) bool {
	for _, name := range f.ReleaseNames {
		if rel.Name == name {
			return true
		}
	}
	return false
}

type OwnerFilter struct {
	ReleaseOwners []string
}

func NewOwnerFilter(ownerNames string) *OwnerFilter {
	return &OwnerFilter{
		ReleaseOwners: strings.Split(ownerNames, ","),
	}
}

func (f *OwnerFilter) Match(rel *v1alpha1.Release) bool {
	if rel.Project == nil {
		return false
	}
	for _, name := range f.ReleaseOwners {
		for _, releaseOwner := range rel.Project.Spec.Owners {
			if strings.Contains(releaseOwner, name) {
				return true
			}
		}
	}
	return false
}
