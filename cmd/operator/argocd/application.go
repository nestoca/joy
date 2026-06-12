// This package is to expose our custom ArgoCD types.
// Unfortunately we cannot import argocd directly since it uses an incompatible version of k8s.io/api with breaking changes.
package argocd

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Application is our definition of the ArgoCD Application CR.
// It is unfortunately impossible to import the argocd project as they depend on old versions of k8s.io/api that have had apis removed.
// Thus we cannot import the project. But defining the type ourselves should not be too much maintenance and allows us to avoid importing the entire kitchen sync
// like in argocd's package design. It's also better than raw yaml with no types at all.
type Application struct {
	metav1.TypeMeta
	metav1.ObjectMeta `json:"metadata"`
	Spec              ApplicationSpec `json:"spec"`
}

type ApplicationSpec struct {
	Project     string                 `json:"project"`
	Source      ApplicationSource      `json:"source"`
	Destination ApplicationDestination `json:"destination"`
	SyncPolicy  SyncPolicy             `json:"syncPolicy,omitzero"`
}

type ApplicationSource struct {
	RepoURL        string          `json:"repoURL"`
	TargetRevision string          `json:"targetRevision"`
	Chart          string          `json:"chart"`
	Helm           SourceHelm      `json:"helm"`
	Directory      SourceDirectory `json:"directory,omitzero"`
}

type SourceHelm struct {
	ReleaseName string `json:"releaseName"`
	Values      string `json:"values"`
}

type SourceDirectory struct {
	// Recurse specifies whether to scan a directory recursively for manifests
	Recurse bool `json:"recurse,omitempty"`
	// Exclude contains a glob pattern to match paths against that should be explicitly excluded from being used during manifest generation
	Exclude string `json:"exclude,omitempty"`
	// Include contains a glob pattern to match paths against that should be explicitly included during manifest generation
	Include string `json:"include,omitempty"`
}

type ApplicationDestination struct {
	Server    string `json:"server"`
	Namespace string `json:"namespace"`
}

type SyncPolicy struct {
	Automated   SyncPolicyAutomated `json:"automated,omitzero"`
	SyncOptions []string            `json:"syncOptions"`
	Retry       SyncPolicyRetry     `json:"retry,omitzero" `
}

type SyncPolicyAutomated struct {
	Prune      *bool `json:"prune,omitzero"`
	SelfHeal   *bool `json:"selfHeal,omitzero"`
	AllowEmpty *bool `json:"allowEmpty,omitzero"`
	Enabled    *bool `json:"enabled,omitzero"`
}

type SyncPolicyRetry struct {
	// Limit is the maximum number of attempts for retrying a failed sync. If set to 0, no retries will be performed.
	Limit int64 `json:"limit,omitempty"`
	// Backoff controls how to backoff on subsequent retries of failed syncs
	Backoff RetryBackoff `json:"backoff,omitzero"`
	// Refresh indicates if the latest revision should be used on retry instead of the initial one (default: false)
	Refresh bool `json:"refresh,omitempty"`
}

type RetryBackoff struct {
	// Duration is the amount to back off. Default unit is seconds, but could also be a duration (e.g. "2m", "1h")
	Duration string `json:"duration,omitempty"`
	// Factor is a factor to multiply the base duration after each failed retry
	Factor *int64 `json:"factor,omitempty"`
	// MaxDuration is the maximum amount of time allowed for the backoff strategy
	MaxDuration string `json:"maxDuration,omitempty"`
}
