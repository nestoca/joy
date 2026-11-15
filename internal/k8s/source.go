package k8s

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	sigsyaml "sigs.k8s.io/yaml"

	"github.com/nestoca/joy/internal/shell"
)

type HelmSource struct {
	AdditionalArgs string `json:"additionalArgs"`
	URL            string `json:"url"`
	Namespace      string `json:"namespace"`
	Version        string `json:"version"`
	Release        string `json:"release"`
	ValuesFile     string `json:"valuesFile"`
	Values         any    `json:"values"`
	KubeContext    string `json:"-"`
}

func (source HelmSource) Apply(ctx context.Context) error {
	ns := cmp.Or(source.Namespace, "default")
	if ns != "default" {
		if err := ensureNamespace(ctx, ns, "orbstack"); err != nil {
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
		"helm upgrade --kube-context %s --create-namespace --namespace %s --install --wait --timeout 5m %s %s %s --values - %s",
		[]any{
			source.KubeContext,
			ns,
			source.Release,
			source.URL,
			func() string {
				if version := source.Version; version != "" {
					return "--version " + version
				}
				return ""
			}(),
			source.AdditionalArgs,
		},
		shell.WithStdin(bytes.NewReader(data)),
	)
}

type KubectlSource struct {
	KubeContext string `json:"-"`
	Path        string `json:"path"`
	Input       string `json:"input"`
	Namespace   string `json:"namespace"`
	Recursive   bool   `json:"recursive"`
}

func (source KubectlSource) Apply(ctx context.Context) (err error) {
	ns := cmp.Or(source.Namespace, "default")
	if ns != "default" {
		if err := ensureNamespace(ctx, ns, "orbstack"); err != nil {
			return fmt.Errorf("failed to ensure namespace %s: %w", ns, err)
		}
	}

	reader := func() io.Reader {
		if source.Input != "" {
			return strings.NewReader(source.Input)
		}
		reader, err := os.Open(source.Path)
		if err != nil {
			return strings.NewReader("")
		}
		return reader
	}()

	decoder := yaml.NewYAMLToJSONDecoder(reader)

	var workloads []unstructured.Unstructured
	for {
		var resource unstructured.Unstructured
		if err := decoder.Decode(&resource); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode kubectl source input: %w", err)
		}
		if resource.GetAPIVersion() == "apps/v1" && slices.Contains([]string{"Deployment", "Statefulset", "Daemonset"}, resource.GetKind()) {
			workloads = append(workloads, resource)
		}
	}

	defer func() {
		if err != nil {
			return
		}
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		var wg sync.WaitGroup
		defer wg.Wait()

		e := make(chan error, len(workloads))

		for _, workload := range workloads {
			wg.Go(func() {
				if err := sh.Execf(
					ctx,
					"kubectl --context %s rollout status -n %s --timeout 5m %s/%s",
					[]any{source.KubeContext, source.Namespace, strings.ToLower(workload.GetKind()), workload.GetName()},
				); err != nil {
					e <- fmt.Errorf("failed to wait for %s/%s to rollout: %w", workload.GetKind(), workload.GetName(), err)
				}
			})
		}

		go func() {
			wg.Wait()
			close(e)
		}()

		err = <-e
	}()

	if source.Input != "" {
		return sh.Execf(
			ctx,
			"kubectl --context %s apply --server-side -n '%s' -f -",
			[]any{source.KubeContext, ns},
			shell.WithStdin(strings.NewReader(source.Input)),
		)
	}
	return sh.Execf(
		ctx,
		"kubectl --context %s apply --server-side -n '%s' -f '%s' %s",
		[]any{source.KubeContext, ns, source.Path, func() string {
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

func ensureNamespace(ctx context.Context, ns, kubectx string) error {
	return sh.Execf(
		ctx,
		`kubectl --context %s apply -f -`,
		[]any{kubectx},
		shell.WithStdin(shell.JSONReader(&corev1.Namespace{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
			ObjectMeta: metav1.ObjectMeta{Name: ns},
		})),
	)
}
