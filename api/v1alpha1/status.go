package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConditionReady is the condition type used to surface the outcome of the
// operator's reconciliation of a resource.
const ConditionReady = "Ready"

// ResourceStatus is the shared status shape for joy resources reconciled by the
// joy-operator. It is populated by the operator (never by the joy CLI) and is
// surfaced as an Argo CD health status via a Lua health check.
//
// The json/yaml tag split on the fields embedding this struct is deliberate:
// encoding/json needs "omitzero" to drop a zero struct, whereas gopkg.in/yaml.v3
// needs "omitempty" (it recurses into structs and does not understand omitzero).
// Both drop an unset status so it never lands in the catalog git repo, while a
// populated status still serializes to the Kubernetes API.
type ResourceStatus struct {
	// ObservedGeneration is the generation of the resource that was last
	// reconciled by the operator.
	ObservedGeneration int64 `json:"observedGeneration,omitempty" yaml:"observedGeneration,omitempty"`

	// Conditions holds the latest observations of the resource's reconcile state.
	Conditions []metav1.Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// StatusObject is the constraint satisfied by pointers to joy resource types
// that expose a ResourceStatus. It lets the operator drive status updates for
// every resource kind through a single generic helper.
type StatusObject[T any] interface {
	*T
	metav1.Object
	GetStatus() *ResourceStatus
}
