package main

import (
	"fmt"
	"os"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/yokecd/yoke/pkg/openapi"
	"go.yaml.in/yaml/v3"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)

	for _, crd := range []apiextv1.CustomResourceDefinition{
		{
			TypeMeta: metav1.TypeMeta{
				APIVersion: apiextv1.SchemeGroupVersion.Identifier(),
				Kind:       "CustomResourceDefinition",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "releases.joy.nesto.ca",
			},
			Spec: apiextv1.CustomResourceDefinitionSpec{
				Group: "joy.nesto.ca",
				Names: apiextv1.CustomResourceDefinitionNames{
					Plural:     "releases",
					Singular:   "release",
					ShortNames: []string{"rel"},
					Kind:       "Release",
					ListKind:   "Releases",
				},
				Scope: apiextv1.NamespaceScoped,
				Versions: []apiextv1.CustomResourceDefinitionVersion{
					{
						Name:    "v1alpha1",
						Served:  true,
						Storage: true,
						Schema: &apiextv1.CustomResourceValidation{
							OpenAPIV3Schema: sanitizeSchema(openapi.SchemaFor[v1alpha1.Release]()),
						},
					},
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				APIVersion: apiextv1.SchemeGroupVersion.Identifier(),
				Kind:       "CustomResourceDefinition",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "environments.joy.nesto.ca",
			},
			Spec: apiextv1.CustomResourceDefinitionSpec{
				Group: "joy.nesto.ca",
				Names: apiextv1.CustomResourceDefinitionNames{
					Plural:     "environments",
					Singular:   "environment",
					ShortNames: []string{"env"},
					Kind:       "Environment",
					ListKind:   "Environments",
				},
				Scope: apiextv1.ClusterScoped,
				Versions: []apiextv1.CustomResourceDefinitionVersion{
					{
						Name:    "v1alpha1",
						Served:  true,
						Storage: true,
						Schema: &apiextv1.CustomResourceValidation{
							OpenAPIV3Schema: sanitizeSchema(openapi.SchemaFor[v1alpha1.Environment]()),
						},
					},
				},
			},
		},
	} {
		crd.SetAnnotations(map[string]string{"helm.sh/resource-policy": "keep"})

		raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&crd)
		if err != nil {
			return fmt.Errorf("failed to convert %s unstructured format: %w", crd.Name, err)
		}

		delete(raw, "status")

		if err := encoder.Encode(raw); err != nil {
			return fmt.Errorf("failed to encode %s: %w", crd.Name, err)
		}
	}

	return nil
}

func sanitizeSchema(schema *apiextv1.JSONSchemaProps) *apiextv1.JSONSchemaProps {
	for _, prop := range []string{"apiVersion", "kind", "metadata"} {
		delete(schema.Properties, prop)
	}
	return schema
}
