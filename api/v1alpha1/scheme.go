package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Group           = "joy.nesto.ca"
	Version         = "v1alpha1"
	ReleaseKind     = "Release"
	EnvironmentKind = "Environment"
	ProjectKind     = "Project"
	CatalogKind     = "Catalog"
)

var GroupVersion = schema.GroupVersion{Group: Group, Version: Version}

var (
	EnvironmentGK = schema.GroupKind{Group: Group, Kind: EnvironmentKind}
	ProjectGK     = schema.GroupKind{Group: Group, Kind: ProjectKind}
	ReleaseGK     = schema.GroupKind{Group: Group, Kind: ReleaseKind}
	CatalogGK     = schema.GroupKind{Group: Group, Kind: CatalogKind}
)

var (
	EnvironmentGVK = schema.GroupVersionKind{Group: Group, Version: Version, Kind: EnvironmentKind}
	ProjectGVK     = schema.GroupVersionKind{Group: Group, Version: Version, Kind: ProjectKind}
	ReleaseGVK     = schema.GroupVersionKind{Group: Group, Version: Version, Kind: ReleaseKind}
	CatalogGVK     = schema.GroupVersionKind{Group: Group, Version: Version, Kind: CatalogKind}
)
