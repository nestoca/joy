package v1alpha1

import "k8s.io/apimachinery/pkg/runtime/schema"

var SchemeGroupVersion = schema.GroupVersion{Group: "joy.nesto.ca", Version: "v1alpha1"}

const (
	KindRelease     = "Release"
	KindEnvironment = "Environment"
	KindProject     = "Project"
)
