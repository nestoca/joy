package k8s

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/davidmdm/x/xerr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nestoca/joy/internal/shell"
)

type ClusterRegistry struct {
	URL             string          `json:"url"`
	AuthCommand     string          `json:"auth"`
	Helm            bool            `json:"helm"`
	ContainerImages bool            `json:"images"`
	TTL             metav1.Duration `json:"ttl"`
}

type Registries map[string]ClusterRegistry

func (registries Registries) Auth(ctx context.Context, kubecontext string) error {
	errs := make([]error, 0, len(registries))
	for name, registry := range registries {
		if err := registry.Auth(ctx, name, kubecontext); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}
	return xerr.JoinOrdered(errs...)
}

func (registry ClusterRegistry) Auth(ctx context.Context, name, kubecontext string) error {
	fmt.Printf("getting credentials for registry: %s\n", name)

	token, err := sh.ExecfCombined(ctx, registry.AuthCommand, nil)
	if err != nil {
		return fmt.Errorf("failed to authenticate with registry: %w", err)
	}

	registryURL, err := url.Parse(registry.URL)
	if err != nil {
		return fmt.Errorf("failed to parse registry url: %w", err)
	}

	dockerConfig, err := json.Marshal(map[string]any{
		"auths": map[string]any{
			registryURL.Host: map[string]any{
				"username": "oauth2accesstoken",
				"password": string(token),
				"auth":     base64.StdEncoding.EncodeToString(append([]byte("oauth2accesstoken:"), token...)),
			},
		},
	})

	if registry.Helm {
		if err := sh.Execf(
			ctx,
			`kubectl --context %s apply -n argocd --server-side -f -`,
			[]any{kubecontext},
			shell.WithStdin(
				shell.JSONReader(
					&corev1.Secret{
						TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
						ObjectMeta: metav1.ObjectMeta{
							Name:   "argocd-helm-" + name,
							Labels: map[string]string{"argocd.argoproj.io/secret-type": "repository"},
						},
						Type:      corev1.SecretTypeOpaque,
						Immutable: new(false),
						StringData: map[string]string{
							"type":      "helm",
							"url":       registryURL.Host,
							"enableOCI": strconv.FormatBool(registryURL.Scheme == "oci:"),
							"username":  "oauth2accesstoken",
							"password":  string(token),
						},
					},
				),
			),
		); err != nil {
			return fmt.Errorf("failed to apply argocd repository helm credentials secret: %w", err)
		}

		if err := sh.Execf(
			ctx,
			`kubectl --context %s apply -n argocd --server-side -f -`,
			[]any{kubecontext},
			shell.WithStdin(
				shell.JSONReader(
					&corev1.Secret{
						TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
						ObjectMeta: metav1.ObjectMeta{
							Name:   "joy-helm-" + name,
							Labels: map[string]string{"joy.nesto.ca/dockerconfigjson": "helm"},
						},
						Type:      corev1.SecretTypeDockerConfigJson,
						Immutable: new(false),
						Data:      map[string][]byte{corev1.DockerConfigJsonKey: dockerConfig},
					},
				),
			),
		); err != nil {
			return fmt.Errorf("failed to apply joy repository helm credentials secret: %w", err)
		}
	}

	if registry.ContainerImages {
		if err != nil {
			return fmt.Errorf("failed to serialize registry dockerconfigjson: %w", err)
		}

		if err := sh.Execf(
			ctx,
			`kubectl --context %s apply --server-side -f -`,
			[]any{kubecontext},
			shell.WithStdin(
				shell.JSONReader(
					&corev1.Secret{
						TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
						ObjectMeta: metav1.ObjectMeta{
							Name: name + "-images",
							// Labels: map[string]string{"argocd.argoproj.io/secret-type": "repository"},
						},
						Type:      corev1.SecretTypeDockerConfigJson,
						Immutable: new(false),
						Data:      map[string][]byte{".dockerconfigjson": dockerConfig},
					},
				),
			),
		); err != nil {
			return fmt.Errorf("failed to apply images dockerconfigjson secret: %w", err)
		}
	}

	return nil
}
