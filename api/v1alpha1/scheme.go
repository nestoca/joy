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
)

var GroupVersion = schema.GroupVersion{Group: Group, Version: Version}

var EnvironmentGK = schema.GroupKind{Group: Group, Kind: EnvironmentKind}
var ProjectGK = schema.GroupKind{Group: Group, Kind: ProjectKind}
var ReleaseGK = schema.GroupKind{Group: Group, Kind: ReleaseKind}

var EnvironmentGVK = schema.GroupVersionKind{Group: Group, Version: Version, Kind: EnvironmentKind}
var ProjectGVK = schema.GroupVersionKind{Group: Group, Version: Version, Kind: ProjectKind}
var ReleaseGVK = schema.GroupVersionKind{Group: Group, Version: Version, Kind: ReleaseKind}
