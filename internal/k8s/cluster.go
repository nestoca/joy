package k8s

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/nestoca/joy/internal/shell"
	"github.com/nestoca/joy/internal/wait"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
)

var sh = shell.Default

type ClusterDefinition struct {
	Env           string                     `json:"-"`
	Recreate      bool                       `json:"-"`
	CRDs          string                     `json:"crds"`
	SetupCommands []string                   `json:"setupCommands"`
	Registries    map[string]ClusterRegistry `json:"registries"`
	Sources       map[string]Source          `json:"sources"`
}

func Setup(ctx context.Context, cluster ClusterDefinition) error {
	if cluster.Recreate {
		fmt.Println("deleting orbstack cluster if present")
		if err := sh.Execf(ctx, "orb delete k8s -f", nil); err != nil {
			return fmt.Errorf("failed to delete local cluster: %w", err)
		}
	}

	fmt.Println("starting orbstack")
	if err := sh.Execf(ctx, "orb start k8s", nil); err != nil {
		return fmt.Errorf("failed to start local cluster: %w", err)
	}

	if context, err := sh.ExecfCombined(ctx, "kubectl config current-context", nil); err != nil {
		return fmt.Errorf("failed to get current kube-context: %w", err)
	} else if string(context) != "orbstack" {
		if err := sh.Execf(ctx, "kubectl config use-context orbstack", nil); err != nil {
			return fmt.Errorf("failed to set kube-context to use orbstack: %w", err)
		}
	}

	if err := wait.Eventually(
		ctx,
		func(ctx context.Context) error {
			return sh.Execf(ctx, "kubectl get ns default", nil)
		},
		wait.WithTicker("testing for cluster readiness"),
	); err != nil {
		return fmt.Errorf("failed to ping cluster")
	}

	for _, cmd := range cluster.SetupCommands {
		if err := sh.Execf(ctx, cmd, nil); err != nil {
			return fmt.Errorf("failed to run: %q: %w", cmd, err)
		}
	}

	if err := ensureNamespace(ctx, "argocd"); err != nil {
		return fmt.Errorf("failed to ensure argocd namespace: %w", err)
	}

	for name, registry := range cluster.Registries {
		if err := func() error {
			token, err := sh.ExecfCombined(ctx, registry.AuthCommand, nil)
			if err != nil {
				return fmt.Errorf("failed to authenticate with registry: %w", err)
			}

			registryURL, err := url.Parse(registry.URL)
			if err != nil {
				return fmt.Errorf("failed to parse registry url: %w", err)
			}

			if registry.Helm {
				if err := sh.Execf(
					ctx,
					`kubectl apply -n argocd --server-side -f -`,
					nil,
					shell.WithStdin(
						shell.JSONReader(
							&corev1.Secret{
								TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
								ObjectMeta: metav1.ObjectMeta{
									Name:   name + "-helm",
									Labels: map[string]string{"argocd.argoproj.io/secret-type": "repository"},
								},
								Type:      corev1.SecretTypeOpaque,
								Immutable: new(false),
								StringData: map[string]string{
									"type": "helm",
									"url":  registryURL.Host,
									"enableOCI": strconv.FormatBool(
										registryURL.Scheme == "oci:",
									),
									"username": "oauth2accesstoken",
									"password": string(token),
								},
							},
						),
					),
				); err != nil {
					return fmt.Errorf("failed to apply helm credentials secret: %w", err)
				}
			}

			if registry.ContainerImages {
				data, err := json.Marshal(map[string]any{
					"auths": map[string]any{
						registryURL.Host: map[string]any{
							"username": "oauth2accesstoken",
							"password": string(token),
							"auth":     base64.StdEncoding.EncodeToString(append([]byte("oauth2accesstoken:"), token...)),
						},
					},
				})
				if err != nil {
					return fmt.Errorf("failed to serialize registry dockerconfigjson: %w", err)
				}

				if err := sh.Execf(
					ctx,
					`kubectl apply --server-side -f -`,
					nil,
					shell.WithStdin(
						shell.JSONReader(
							&corev1.Secret{
								TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
								ObjectMeta: metav1.ObjectMeta{
									Name:   name + "-images",
									Labels: map[string]string{"argocd.argoproj.io/secret-type": "repository"},
								},
								Type:      corev1.SecretTypeDockerConfigJson,
								Immutable: new(false),
								Data:      map[string][]byte{".dockerconfigjson": data},
							},
						),
					),
				); err != nil {
					return fmt.Errorf("failed to apply images dockerconfigjson secret: %w", err)
				}
			}

			return nil
		}(); err != nil {
			return fmt.Errorf("failed to setup registry %q: %w", name, err)
		}
	}

	crds := KubectlSource{
		Path:      cluster.CRDs,
		Recursive: true,
	}

	if err := wait.TickFunc(ctx, "applying crds", crds.Apply); err != nil {
		return fmt.Errorf("failed to apply crds: %w", err)
	}

	argocd := HelmSource{
		URL:       "https://github.com/argoproj/argo-helm/releases/download/argo-cd-10.9.0/argo-cd-10.9.0.tgz",
		Namespace: "argocd",
		Release:   "argo-cd",
		Values: strings.NewReader(`
      configs:
        params:
          server.insecure: true
        cm:
          users.anonymous.enabled: "true"
        rbac:
          policy.default: role:admin
      `),
	}

	if err := wait.TickFunc(ctx, "applying argocd", argocd.Apply); err != nil {
		return fmt.Errorf("failed to apply argocd: %w", err)
	}

	joyOperatorVersion := "0.7.2"

	joyOperator := HelmSource{
		URL:       "oci://ghcr.io/nestoca/joy-operator-chart",
		Namespace: "argocd",
		Version:   joyOperatorVersion,
		Release:   "joy-operator",
		Values: map[string]any{
			"helm": func() map[string]any {
				if len(cluster.Registries) == 0 {
					return nil
				}
				for name, registry := range cluster.Registries {
					if !registry.Helm {
						continue
					}
					return map[string]any{
						"registry": registry.URL,
						"user":     "oauth2accesstoken",
						"credentials": map[string]any{
							"secret":    name + "-helm",
							"key":       "password",
							"mountPath": "/secrets/registry.creds",
						},
					}
				}
				return nil
			}(),
			"environmentDestinations": map[string]any{
				cluster.Env: map[string]any{
					"server":    "https://kubernetes.default.svc",
					"namespace": "default",
				},
			},
			"installCrds": true,
			"version":     joyOperatorVersion,
		},
	}

	if err := wait.TickFunc(ctx, "applying joy-operator", joyOperator.Apply); err != nil {
		return fmt.Errorf("failed to apply joy-operator: %w", err)
	}

	for name, source := range cluster.Sources {
		if err := wait.TickFunc(ctx, "applying "+name, source.Apply); err != nil {
			return fmt.Errorf("failed to apply %s: %w", name, err)
		}
	}

	fmt.Println("cluster setup and operational")

	return nil
}
