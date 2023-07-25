package secret

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/TwiN/go-color"
	"github.com/nestoca/joy/internal/catalog"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/yml"
	"os/exec"
	"strings"
)

func ImportCert() error {
	err := ensureKubectlInstalled()
	if err != nil {
		return err
	}

	// Select kube context
	context, err := selectKubeContext()
	if err != nil {
		return fmt.Errorf("selecting kube context: %w", err)
	}

	// Fetch sealed secret public certificate
	cmd := exec.Command("kubectl",
		"--context", context,
		"get", "secret",
		"-n", "sealed-secrets",
		"-l", "sealedsecrets.bitnami.com/sealed-secrets-key=active",
		"-o", "jsonpath={@.items[0].data['tls\\.crt']}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("running kubectl command to fetch sealed secret public certificate: %w", err)
	}

	// Decode base64 encoded certificate
	cert, err := base64.StdEncoding.DecodeString(string(output))
	if err != nil {
		return fmt.Errorf("decoding base64 encoded certificate: %w", err)
	}

	// Load catalog
	loadOpts := catalog.LoadOpts{
		LoadEnvs:        true,
		SortEnvsByOrder: true,
	}
	cat, err := catalog.Load(loadOpts)
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	// Select environment
	selectedEnv, err := selectEnvironment(cat.Environments)
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

	fmt.Printf(`✅ Imported sealed secrets public certificate from cluster %s into environment %s
Make sure to commit and push those changes to git.
`,
		color.InYellow(context),
		color.InYellow(selectedEnv.Name))
	return nil
}

func selectEnvironment(environments []*environment.Environment) (*environment.Environment, error) {
	// Create list of environment names
	var envNames []string
	for _, env := range environments {
		envNames = append(envNames, env.Name)
	}

	// Prompt user to select environment
	var selectedIndex int
	err := survey.AskOne(&survey.Select{
		Message: "Select environment to import sealed secrets public certificate into",
		Options: envNames,
	},
		&selectedIndex,
		survey.WithPageSize(10),
		survey.WithValidator(survey.Required),
	)
	if err != nil {
		return nil, fmt.Errorf("prompting for environment: %w", err)
	}
	return environments[selectedIndex], nil
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
		Message: "Select kube context of cluster to fetch seal secrets public certificate from",
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

func ensureKubectlInstalled() error {
	cmd := exec.Command("which", "kubectl")
	err := cmd.Run()
	if err != nil {
		fmt.Println("🤓 This command requires kubectl cli to be installed: https://kubernetes.io/docs/tasks/tools/#kubectl")
		return errors.New("missing kubectl cli dependency")
	}
	return nil
}
