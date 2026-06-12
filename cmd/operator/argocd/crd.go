package argocd

import (
	_ "embed"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

//go:embed apps.crd.yaml
var rawApplicationCRD []byte

var ApplicationCRD = func() *apiextensionsv1.CustomResourceDefinition {
	var crd apiextensionsv1.CustomResourceDefinition
	if err := yaml.Unmarshal(rawApplicationCRD, &crd); err != nil {
		panic(fmt.Errorf("failed to decode argocd Application CRD: %w", err))
	}
	return &crd
}()
