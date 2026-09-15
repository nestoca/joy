package k8s

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nestoca/joy/internal/shell"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

type ClusterRegistry struct {
	URL             string `json:"url"`
	AuthCommand     string `json:"auth"`
	Helm            bool   `json:"helm"`
	ContainerImages bool   `json:"images"`
}

type HelmSource struct {
	URL        string `json:"url"`
	Namespace  string `json:"namespace"`
	Version    string `json:"version"`
	Release    string `json:"release"`
	ValuesFile string `json:"valuesFile"`
	Values     any    `json:"values"`
}

func (source HelmSource) Apply(ctx context.Context) error {
	ns := cmp.Or(source.Namespace, "default")
	if ns != "default" {
		if err := ensureNamespace(ctx, ns); err != nil {
			return fmt.Errorf("failed to ensure namespace %s: %w", ns, err)
		}
	}
	data, err := func() ([]byte, error) {
		if value := source.Values; value != nil {
			if reader, ok := source.Values.(io.Reader); ok {
				return io.ReadAll(reader)
			}
			if data, ok := source.Values.(string); ok {
				return []byte(data), nil
			}
			return sigsyaml.Marshal(value)
		}
		return os.ReadFile(source.ValuesFile)
	}()
	if err != nil {
		return fmt.Errorf("failed to get values: %w", err)
	}
	return sh.Execf(
		ctx,
		"helm upgrade --create-namespace --namespace %s --install --wait --timeout 5m %s %s %s --values -",
		[]any{
			ns,
			source.Release,
			source.URL,
			func() string {
				if version := source.Version; version != "" {
					return "--version " + version
				}
				return ""
			}(),
		},
		shell.WithStdin(bytes.NewReader(data)),
	)
}

type KubectlSource struct {
	Path      string `json:"path"`
	Input     string `json:"input"`
	Namespace string `json:"namespace"`
	Recursive bool   `json:"recursive"`
}

func (source KubectlSource) Apply(ctx context.Context) error {
	ns := cmp.Or(source.Namespace, "default")
	if ns != "default" {
		if err := ensureNamespace(ctx, ns); err != nil {
			return fmt.Errorf("failed to ensure namespace %s: %w", ns, err)
		}
	}
	if source.Input != "" {
		return sh.Execf(
			ctx,
			"kubectl apply --server-side -n '%s' -f -",
			[]any{ns},
			shell.WithStdin(strings.NewReader(source.Input)),
		)
	}
	return sh.Execf(
		ctx,
		"kubectl apply --server-side -n '%s' -f '%s' %s",
		[]any{ns, source.Path, func() string {
			if source.Recursive {
				return "--recursive"
			}
			return ""
		}()},
	)
}

type Source struct {
	Helm    *HelmSource    `json:"helm"`
	Kubectl *KubectlSource `json:"kubectl"`
}

func (source Source) Apply(ctx context.Context) error {
	if source.Helm != nil {
		return source.Helm.Apply(ctx)
	}
	if source.Kubectl != nil {
		return source.Kubectl.Apply(ctx)
	}
	return nil
}

func ensureNamespace(ctx context.Context, ns string) error {
	return sh.Execf(
		ctx,
		`kubectl apply -f -`,
		nil,
		shell.WithStdin(shell.JSONReader(&corev1.Namespace{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
			ObjectMeta: metav1.ObjectMeta{Name: ns},
		})),
	)
}
