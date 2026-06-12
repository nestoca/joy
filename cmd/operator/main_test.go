package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/nestoca/joy/cmd/operator/argocd"
	"github.com/yokecd/yoke/pkg/k8s"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
)

func TestMain(m *testing.M) {
	if run, _ := strconv.ParseBool(os.Getenv("RUN_OPERATOR_TESTS")); !run {
		os.Exit(0)
	}

	if err := withStdio(exec.Command("kind", "delete", "cluster", "--name=joy-operator")).Run(); err != nil {
		panic(err)
	}

	if err := withStdio(exec.Command("kind", "create", "cluster", "--name=joy-operator")).Run(); err != nil {
		panic(err)
	}

	if err := withStdio(exec.Command("docker", "build", "--tag=joy-operator:test", "../..")).Run(); err != nil {
		panic(err)
	}

	if err := withStdio(exec.Command("kind", "load", "docker-image", "joy-operator:test", "--name=joy-operator")).Run(); err != nil {
		panic(err)
	}

	if err := withStdio(
		exec.Command(
			"helm",
			"install",
			"joy-operator",
			"../../chart",
			"--set=image=joy-operator",
			"--set=version=test",
			"--set=argocd.namespace=default",
		),
	).Run(); err != nil {
		panic(err)
	}

	client, err := getKubeClient()
	if err != nil {
		panic(err)
	}

	crdIntf := k8s.TypedInterface[apiextensionsv1.CustomResourceDefinition](client.Dynamic, schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	})

	if _, err := crdIntf.Apply(context.Background(), argocd.ApplicationCRD, metav1.ApplyOptions{FieldManager: "operator-tests"}); err != nil {
		panic(err)
	}

	fmt.Println("waiting for application crd to become ready")
	// TODO: when yoke exposes its WaitForReady function on the kubernetes client we could use that instead.
	// For now we can use a lazy sleep style.
	time.Sleep(2 * time.Second)

	os.Exit(m.Run())
}

func withStdio(cmd *exec.Cmd) *exec.Cmd {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd
}

func getKubeClient() (*k8s.Client, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube/config"))
	if err != nil {
		return nil, fmt.Errorf("failed to construct kuberentes rest config: %w", err)
	}
	return k8s.NewClient(cfg)
}
