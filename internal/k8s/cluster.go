package k8s

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	sigsyaml "sigs.k8s.io/yaml"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/davidmdm/x/xcontainer"
	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/shell"
	"github.com/nestoca/joy/internal/wait"
	"github.com/nestoca/joy/pkg/catalog"
)

var (
	sh        = shell.Default
	withStdio = []shell.ExecOption{shell.WithStdin(os.Stdin), shell.WithStdout(os.Stdout), shell.WithStderr(os.Stderr)}
)

type ClusterDefinition struct {
	KubeContext      string                `json:"kubeContext"`
	Recreate         bool                  `json:"-"`
	CRDs             string                `json:"crds"`
	SetupCommands    []SetupCommand        `json:"setupCommands"`
	Registries       Registries            `json:"registries"`
	Sources          map[string]Source     `json:"sources"`
	JoyOperator      JoyOperatorDefinition `json:"joyOperator"`
	RequiredProjects []string              `json:"requiredProjects"`
	Env              *v1alpha1.Environment `json:"-"`
	Cat              *catalog.Catalog      `json:"-"`
}

type JoyOperatorDefinition struct {
	Image   string `json:"image"`
	Version string `json:"version"`
	Chart   string `json:"chart"`
}

type SetupCommand struct {
	Value        string `json:"cmd"`
	InheritStdio bool   `json:"inheritStdio"`
}

func (cmd *SetupCommand) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err == nil {
		cmd.Value = raw
		return nil
	}
	type alt SetupCommand
	return json.Unmarshal(data, (*alt)(cmd))
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
			return sh.Execf(ctx, "kubectl get --context %s ns default", []any{cluster.KubeContext})
		},
		wait.WithTicker("testing for cluster readiness"),
	); err != nil {
		return fmt.Errorf("failed to ping cluster: %w", err)
	}

	for _, cmd := range cluster.SetupCommands {
		options := func() []shell.ExecOption {
			if !cmd.InheritStdio {
				return nil
			}
			return withStdio
		}()
		if err := sh.Execf(ctx, cmd.Value, nil, options...); err != nil {
			return fmt.Errorf("failed to run:\n%s\n\n%w", cmd.Value, err)
		}
	}

	if err := ensureNamespace(ctx, "argocd", "orbstack"); err != nil {
		return fmt.Errorf("failed to ensure argocd namespace: %w", err)
	}

	if err := cluster.Registries.Auth(ctx, cluster.KubeContext); err != nil {
		return fmt.Errorf("failed to authenticate registries: %w", err)
	}

	crds := KubectlSource{
		KubeContext: cluster.KubeContext,
		Path:        cluster.CRDs,
		Recursive:   true,
	}

	if err := wait.TickFunc(ctx, "applying crds", crds.Apply); err != nil {
		return fmt.Errorf("failed to apply crds: %w", err)
	}

	if cluster.Sources == nil {
		cluster.Sources = map[string]Source{}
	}

	cluster.Sources["argocd"] = Source{
		Helm: &HelmSource{
			URL:         "https://github.com/argoproj/argo-helm/releases/download/argo-cd-10.9.0/argo-cd-10.9.0.tgz",
			Namespace:   "argocd",
			Release:     "argo-cd",
			KubeContext: cluster.KubeContext,
			Values: strings.NewReader(`
        configs:
          params:
            server.insecure: true
          cm:
            users.anonymous.enabled: "true"
            resource.customizations.health.gateway.networking.k8s.io_HTTPRoute: |
              hs = {}
              hs.status = "Healthy"
              hs.message = "HTTPRoute health checks are disabled for local clusters"
              return hs
            resource.customizations.health.keda.sh_ScaledObject: |
              hs = {}
              hs.status = "Healthy"
              hs.message = "ScaledObject health checks are disabled for local clusters"
              return hs
          rbac:
            policy.default: role:admin
      `),
		},
	}

	joyOperatorVersion := cmp.Or(cluster.JoyOperator.Version, "latest")

	cluster.Sources["joy-operator"] = Source{
		Helm: &HelmSource{
			URL:         cmp.Or(cluster.JoyOperator.Chart, "oci://ghcr.io/nestoca/joy-operator-chart"),
			Namespace:   "argocd",
			KubeContext: cluster.KubeContext,
			Version:     joyOperatorVersion,
			Release:     "joy-operator",
			Values: map[string]any{
				"image":   cmp.Or(cluster.JoyOperator.Image, "ghcr.io/nestoca/joy-operator"),
				"version": joyOperatorVersion,
				"environmentDestinations": map[string]any{
					cluster.Env.Name: map[string]any{
						"server":    "https://kubernetes.default.svc",
						"namespace": "default",
					},
				},
				"installCrds": true,
			},
		},
	}

	type Result struct {
		Name    string
		Err     error
		Elapsed time.Duration
	}
	results := make(chan Result, len(cluster.Sources))
	var wg sync.WaitGroup

	fmt.Println("applying", strings.Join(slices.Sorted(maps.Keys(cluster.Sources)), ", "))

	for name, source := range cluster.Sources {
		wg.Go(func() {
			if source.Helm != nil {
				source.Helm.KubeContext = cluster.KubeContext
			}
			if source.Kubectl != nil {
				source.Kubectl.KubeContext = cluster.KubeContext
			}

			start := time.Now()
			err := source.Apply(ctx)
			elapsed := time.Since(start)

			results <- Result{
				Name:    name,
				Err:     err,
				Elapsed: elapsed.Round(100 * time.Millisecond),
			}
		})
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var done int

	var failedSetup bool
	for result := range results {
		done++
		if result.Err != nil {
			failedSetup = true
			fmt.Printf("  (%d/%d) %s: failed to apply after %s: %v\n", done, len(cluster.Sources), result.Name, result.Elapsed, result.Err)
		} else {
			fmt.Printf("  (%d/%d) %s: succeeded after %s\n", done, len(cluster.Sources), result.Name, result.Elapsed)
		}
	}

	if failedSetup {
		return fmt.Errorf("failed to setup cluster dependencies")
	}

	if requiredProjects := xcontainer.ToSet(cluster.RequiredProjects); len(requiredProjects) > 0 {
		fmt.Println("setting up required projects")

		catalogRelative := func(value string) string {
			result, _ := filepath.Rel(cluster.Cat.Dir, value)
			return result
		}

		files := []string{
			"catalog.yaml",
			catalogRelative(filepath.Join(cluster.Env.Dir, "env.yaml")),
		}

		var projects []string
		for _, proj := range cluster.Cat.Projects {
			if requiredProjects.Has(proj.Name) {
				files = append(files, catalogRelative(proj.File.Path))
				projects = append(projects, proj.Name)
			}
		}

		if missingProjects := requiredProjects.Difference(xcontainer.ToSet(projects)); len(missingProjects) > 0 {
			return fmt.Errorf("required projects not found: %s", strings.Join(missingProjects.Collect(), ", "))
		}

		var releases []string
		for _, cross := range cluster.Cat.Releases.Items {
			for _, rel := range cross.Releases {
				if rel == nil || rel.Project == nil || !requiredProjects.Has(rel.Project.Name) {
					continue
				}
				files = append(files, catalogRelative(rel.File.Path))
				releases = append(releases, rel.Name)
			}
		}

		DeployResources(ctx, DeployResourcesParams{
			Files:   asClosedChan(files),
			Cluster: cluster,
		})

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		e := make(chan error, len(releases))

		fmt.Println("waiting for required project releases to become ready:", strings.Join(releases, ", "))

		var wg sync.WaitGroup
		for _, release := range releases {
			wg.Go(func() {
				var deployment string
				if err := wait.Eventually(
					ctx,
					func(ctx context.Context) error {
						output, err := sh.ExecfCombined(ctx, "kubectl --context %s get deployment -l release=%s -o name", []any{cluster.KubeContext, release})
						if err != nil {
							return err
						}
						fields := strings.Fields(string(output))
						if len(fields) == 0 {
							return fmt.Errorf("no deployment for release found")
						}
						for _, suffix := range []string{"-api", "-frontend", "-redis"} {
							for _, field := range fields {
								if strings.HasSuffix(field, suffix) {
									deployment = field
									return nil
								}
							}
						}
						return nil
					},
					wait.WithInterval(time.Second),
					wait.WithTimeout(time.Minute),
				); err != nil {
					e <- fmt.Errorf("error waiting release %s deployment to be created: %w", release, err)
					return
				}
				if deployment == "" {
					e <- fmt.Errorf("failed to find corresponding api/frontend/redis deployment for release: %s", release)
					return
				}

				fmt.Printf("found release %s deployment: %s: waiting for it to become healthy\n", release, deployment)

				if err := sh.Execf(ctx, "kubectl --context %s rollout status %s", []any{cluster.KubeContext, deployment}); err != nil {
					e <- fmt.Errorf("failed to watch release %s deployment become healthy: %w", release, err)
				}
			})
		}

		go func() {
			wg.Wait()
			close(e)
		}()

		if err := <-e; err != nil {
			return fmt.Errorf("failed to wait for releases to become ready: %w", err)
		}

	}

	fmt.Println("cluster setup and operational")

	return nil
}

type DeployResourcesParams struct {
	Files   <-chan string
	Cluster ClusterDefinition
}

func DeployResources(ctx context.Context, params DeployResourcesParams) {
	var wg sync.WaitGroup
	defer wg.Wait()

	ensureNamespace(ctx, params.Cluster.Env.Name, "orbstack")

	sh := sh.Dir(params.Cluster.Cat.Dir)

	for range 8 {
		wg.Go(func() {
			for path := range params.Files {
				err := func() error {
					data, err := os.ReadFile(filepath.Join(params.Cluster.Cat.Dir, path))
					if err != nil {
						return err
					}
					var resource metav1.TypeMeta
					if err := sigsyaml.Unmarshal(data, &resource); err != nil {
						return err
					}

					var nsOption string
					if resource.Kind == "Release" {
						nsOption = "-n " + params.Cluster.Env.Name
					}

					return sh.Execf(
						ctx,
						"kubectl --context %s apply -f %s %s",
						[]any{params.Cluster.KubeContext, path, nsOption},
					)
				}()
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to process %q: %v\n", path, err)
					continue
				}
				fmt.Fprintf(os.Stderr, "applied %s\n", path)
			}
		})
	}
}

func asClosedChan[T any](list []T) <-chan T {
	c := make(chan T, len(list))
	for _, value := range list {
		c <- value
	}
	close(c)
	return c
}
