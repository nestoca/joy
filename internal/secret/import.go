package secret

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"

	"github.com/AlecAivazis/survey/v2"

	"github.com/nestoca/joy/internal/dependencies"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/catalog"
)

var kubectlDependency = &dependencies.Dependency{
	Command:    "kubectl",
	Url:        "https://kubernetes.io/docs/tasks/tools/#kubectl",
	IsRequired: false,
	RequiredBy: []string{"sealed-secret import"},
}

func init() {
	dependencies.Add(kubectlDependency)
}

func ImportCert() error {
	if err := kubectlDependency.MustBeInstalled(); err != nil {
		return err
	}

	// Select kube context
	context, err := selectKubeContext()
	if err != nil {
		return fmt.Errorf("selecting kube context: %w", err)
	}

	// Fetch sealed secret certificate
	cmd := exec.Command("kubectl",
		"--context", context,
		"get", "secret",
		"-n", "sealed-secrets",
		"-l", "sealedsecrets.bitnami.com/sealed-secrets-key=active",
		"-o", "jsonpath={@.items[0].data['tls\\.crt']}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("running kubectl command to fetch sealed secret certificate: %w", err)
	}

	// Decode base64 encoded certificate
	cert, err := base64.StdEncoding.DecodeString(string(output))
	if err != nil {
		return fmt.Errorf("decoding base64 encoded certificate: %w", err)
	}

	cat, err := catalog.Load(catalog.LoadOpts{SortEnvsByOrder: true})
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	// Select environment
	selectedEnv, err := environment.SelectSingle(
		cat.Environments,
		nil,
		"Select environment to import sealed secrets certificate into")
	if err != nil {
		return fmt.Errorf("selecting environment: %w", err)
	}

	// Update environment
	err = yml.SetOrAddNodeValue(selectedEnv.File.Tree, "spec.sealedSecretsCert", string(cert))
	if err != nil {
		return fmt.Errorf("updating environment sealed secrets cert node value: %w", err)
	}
	err = selectedEnv.File.UpdateYamlFromTree()
	if err != nil {
		return fmt.Errorf("updating environment yaml from tree: %w", err)
	}
	err = selectedEnv.File.WriteYaml()
	if err != nil {
		return fmt.Errorf("writing environment yaml: %w", err)
	}

	fmt.Printf(`✅ Imported sealed secrets certificate from cluster %s into environment %s
Make sure to commit and push those changes to git.
`,
		style.Resource(context),
		style.Resource(selectedEnv.Name))
	return nil
}

func selectKubeContext() (string, error) {
	// Call `kubectl` to get the list of contexts
	cmd := exec.Command("kubectl", "config", "get-contexts", "-o", "name")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("running kubectl command to get list of contexts: %w", err)
	}
	contexts := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Prompt user to select a context
	var selectedIndex int
	err = survey.AskOne(&survey.Select{
		Message: "Select kube context of cluster to fetch seal secrets certificate from",
		Options: contexts,
	},
		&selectedIndex,
		survey.WithPageSize(10),
		survey.WithValidator(survey.Required),
	)
	if err != nil {
		return "", fmt.Errorf("prompting for kube context: %w", err)
	}
	return contexts[selectedIndex], nil
}
