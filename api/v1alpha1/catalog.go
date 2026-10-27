package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	sigsyaml "sigs.k8s.io/yaml"

	"github.com/nestoca/joy/internal/helm"
	"github.com/nestoca/joy/internal/yml"
)

type Catalog struct {
	metav1.TypeMeta   `yaml:",inline"`
	metav1.ObjectMeta `json:"metadata" yaml:"metadata"`
	Spec              CatalogSpec    `json:"spec" yaml:"spec"`
	Status            ResourceStatus `json:"status,omitzero" yaml:"status,omitempty"`
	File              *yml.File      `json:"-" yaml:"-"`
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

func CatalogFromFile(file *yml.File) (*Catalog, error) {
	data, err := file.Yaml()
	if err != nil {
		return nil, err
	}
	var catalog Catalog
	if err := sigsyaml.UnmarshalStrict(data, &catalog); err != nil {
		return nil, err
	}
	catalog.File = file
	return &catalog, nil
}

func IsValidCatalog(apiVersion, kind string) bool {
	return apiVersion == GroupVersion.Identifier() && kind == CatalogKind
}
