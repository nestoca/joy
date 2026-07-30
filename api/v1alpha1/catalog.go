package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nestoca/joy/internal/helm"
)

type Catalog struct {
	metav1.TypeMeta   `yaml:",inline"`
	metav1.ObjectMeta `json:"metadata" yaml:"metadata"`
	Spec              CatalogSpec    `json:"spec" yaml:"spec"`
	Status            ResourceStatus `json:"status,omitzero" yaml:"status,omitempty"`
}

func (c *Catalog) GetStatus() *ResourceStatus { return &c.Status }

type CatalogSpec struct {
	RepoURL  string        `json:"repoUrl" yaml:"repoUrl"`
	Revision string        `json:"revision" yaml:"revision"`
	Charts   CatalogCharts `json:"charts,omitzero" yaml:"charts,omitempty"`
}

type CatalogCharts struct {
	Default string                `json:"default" yaml:"default"`
	Refs    map[string]helm.Chart `json:"refs" yaml:"refs"`
}
