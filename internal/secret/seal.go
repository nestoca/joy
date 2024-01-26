package secret

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/AlecAivazis/survey/v2"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/pkg/catalog"
)

type SealOptions struct {
	Env         string
	InputIsTTY  bool
	OutputIsTTY bool
}

func Seal(cat *catalog.Catalog, opts SealOptions) error {
	environment, err := getEnvironment(cat.Environments, opts.Env)
	if err != nil {
		return err
	}

	// Get sealed secrets certificate
	cert := environment.Spec.SealedSecretsCert
	if cert == "" {
		return fmt.Errorf("🤷 Environment %s has no sealed secrets certificate configured, please run `joy secrets import` first", style.Resource(environment.Name))
	}

	// Write cert to temporary file
	certFile, err := writeStringToTempFile(cert)
	if err != nil {
		return fmt.Errorf("writing certificate to temporary file: %w", err)
	}
	defer os.Remove(certFile)

	secret, err := func() (io.Reader, error) {
		if opts.InputIsTTY {
			return readFromTTY()
		}
		return os.Stdin, nil
	}()
	if err != nil {
		return err
	}

	output, err := seal(secret, certFile)
	if err != nil {
		return fmt.Errorf("running kubeseal command: %w", err)
	}

	if !opts.OutputIsTTY {
		fmt.Println(output)
		return nil
	}

	fmt.Println("🔒 Sealed secret:")
	fmt.Println(style.Code(output))
	return nil
}

func readFromTTY() (io.Reader, error) {
	// Temporarily tweak multiline question template to start editing on new line and also hide final answer
	oldTemplate := survey.MultilineQuestionTemplate
	survey.MultilineQuestionTemplate = `
{{- if .ShowHelp }}{{- color .Config.Icons.Help.Format }}{{ .Config.Icons.Help.Text }} {{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color .Config.Icons.Question.Format }}{{ .Config.Icons.Question.Text }} {{color "reset"}}
{{- color "default+hb"}}{{ .Message }} {{color "reset"}}
{{- if .ShowAnswer}}
{{else}}
{{- if .Default}}{{color "white"}}({{.Default}}) {{color "reset"}}{{end}}
{{- color "cyan"}}[Enter 2 empty lines to finish]{{color "reset"}}
{{end}}`
	defer func() {
		survey.MultilineQuestionTemplate = oldTemplate
	}()

	// Prompt user for multiline secret input
	var secret string

	if err := survey.AskOne(
		&survey.Multiline{Message: "Enter secret to seal"},
		&secret, survey.WithValidator(survey.Required),
		survey.WithHideCharacter('*'),
	); err != nil {
		return nil, fmt.Errorf("prompting for secret: %w", err)
	}

	return strings.NewReader(strings.TrimSpace(secret)), nil
}

// Seal secret by running `kubeseal --raw --scope cluster-wide --cert <cert>`
func seal(r io.Reader, certFile string) (string, error) {
	// Seal secret by running `kubeseal --raw --scope cluster-wide --cert <cert>`
	cmd := exec.Command("kubeseal",
		"--raw",
		"--scope", "cluster-wide",
		"--cert", certFile)

	var buffer bytes.Buffer

	cmd.Stdin = r
	cmd.Stdout = &buffer

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run kubeseal command: %w", err)
	}

	return buffer.String(), nil
}

func writeStringToTempFile(text string) (string, error) {
	f, err := os.CreateTemp("", "joy-")
	if err != nil {
		return "", fmt.Errorf("creating temporary file: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(text)
	if err != nil {
		return "", fmt.Errorf("writing to temporary file: %w", err)
	}

	return f.Name(), nil
}

func getEnvironment(environments []*v1alpha1.Environment, name string) (*v1alpha1.Environment, error) {
	if name == "" {
		return environment.SelectSingle(environments, nil, "Select environment to seal secret in")
	}

	selectedEnv := environment.FindByName(environments, name)
	if selectedEnv == nil {
		return nil, fmt.Errorf("environment %s not found", name)
	}

	return selectedEnv, nil
}
