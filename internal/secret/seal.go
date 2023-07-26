package secret

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/nestoca/joy/internal/catalog"
	"github.com/nestoca/joy/internal/environment"
	"github.com/nestoca/joy/internal/style"
	"os"
	"os/exec"
	"strings"
)

func Seal(env string) error {
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
	var selectedEnv *environment.Environment
	if env != "" {
		selectedEnv = environment.FindByName(cat.Environments, env)
	}
	selectedEnv, err = environment.SelectSingle(
		cat.Environments,
		selectedEnv,
		"Select environment to seal secret in")
	if err != nil {
		return err
	}

	// Get sealed secrets certificate
	cert := selectedEnv.Spec.SealedSecretsCert
	if cert == "" {
		fmt.Printf("🤷 Environment %s has no sealed secrets certificate configured, please run `joy secrets import` first.\n", style.Resource(selectedEnv.Name))
		return nil
	}

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
	err = survey.AskOne(
		&survey.Multiline{
			Message: "Enter secret to seal",
		},
		&secret,
		survey.WithValidator(survey.Required),
		survey.WithHideCharacter('*'),
	)
	if err != nil {
		return fmt.Errorf("prompting for secret: %w", err)
	}
	secret = strings.TrimSpace(secret)

	// Write cert to temporary file
	certFile, err := writeStringToTempFile(cert)
	if err != nil {
		return fmt.Errorf("writing certificate to temporary file: %w", err)
	}
	defer os.Remove(certFile)

	// Seal secret by running `kubeseal --raw --scope cluster-wide --cert <cert>`
	cmd := exec.Command("kubeseal",
		"--raw",
		"--scope", "cluster-wide",
		"--cert", certFile)
	cmd.Stdin = strings.NewReader(secret)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("running kubeseal command: %w", err)
	}

	// Print sealed secret
	fmt.Println("🔒 Sealed secret:")
	fmt.Println(style.Code(string(output)))
	return nil
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
