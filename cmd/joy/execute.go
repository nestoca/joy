package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nestoca/survey/v2"
	"github.com/spf13/cobra"

	"github.com/nestoca/joy/internal/config"
)

func NewExecuteCmd() *cobra.Command {
	var list bool
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:     "execute",
		Aliases: []string{"exec", "x"},
		Args:    cobra.ArbitraryArgs,
		Short:   "Execute user joy scripts",
		Long:    `Execute custom user joy scripts.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			searchPath := os.Getenv("PATH")

			scripts, err := getScripts(cfg.CatalogDir, searchPath)
			if err != nil {
				return fmt.Errorf("getting scripts: %w", err)
			}

			if list {
				if jsonOutput {
					output, err := formatScriptsAsJson(scripts)
					if err != nil {
						return fmt.Errorf("formatting scripts as JSON: %w", err)
					}
					_, err = fmt.Fprintln(cmd.OutOrStdout(), output)
					return err
				}

				_, err = fmt.Fprintln(cmd.OutOrStdout(), formatScripts(scripts))
				return err
			}

			script, err := func() (Script, error) {
				if len(args) == 0 {
					return selectScript(scripts)
				}
				return getScript(scripts, args[0])
			}()
			if err != nil {
				return fmt.Errorf("getting/selecting script: %w", err)
			}

			if len(args) > 0 {
				args = args[1:]
			}

			return executeScript(cfg.CatalogDir, script.Path, args)
		},
	}

	cmd.Flags().BoolVarP(&list, "list", "l", false, "List available scripts")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

func selectScript(scripts []Script) (Script, error) {
	var scriptNames []string
	for _, script := range scripts {
		scriptNames = append(scriptNames, script.Name)
	}
	prompt := &survey.Select{
		Message: "Select script to execute:",
		Options: scriptNames,
	}
	var index int
	if err := survey.AskOne(prompt, &index); err != nil {
		return Script{}, fmt.Errorf("prompting for script selection: %w", err)
	}
	return scripts[index], nil
}

type Script struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func getScripts(catalogDir, searchPath string) ([]Script, error) {
	scriptsMap := make(map[string]Script)

	catalogBinDir := filepath.Join(catalogDir, "bin")
	files, err := os.ReadDir(catalogBinDir)
	if err != nil {
		return nil, fmt.Errorf("listing files under catalog bin dir %q: %w", catalogBinDir, err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		scriptsMap[name] = Script{
			Name: name,
			Path: filepath.Join(catalogBinDir, file.Name()),
		}
	}

	searchDirs := filepath.SplitList(searchPath)
	for _, dir := range searchDirs {
		files, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("listing files under search dir %q: %w", dir, err)
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			name := file.Name()
			if !strings.HasPrefix(name, "joy-") {
				continue
			}
			nameWithoutPrefix := strings.TrimPrefix(name, "joy-")
			scriptsMap[nameWithoutPrefix] = Script{
				Name: nameWithoutPrefix,
				Path: filepath.Join(dir, name),
			}
		}
	}

	var scripts []Script
	for _, script := range scriptsMap {
		scripts = append(scripts, script)
	}
	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].Name < scripts[j].Name
	})

	return scripts, nil
}

func formatScripts(scripts []Script) string {
	var names []string
	for _, script := range scripts {
		names = append(names, script.Name)
	}
	return strings.Join(names, "\n")
}

func formatScriptsAsJson(scripts []Script) (string, error) {
	b, err := json.MarshalIndent(scripts, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshalling scripts to JSON: %w", err)
	}
	return string(b), nil
}

func getScript(scripts []Script, scriptName string) (Script, error) {
	for _, script := range scripts {
		if script.Name == scriptName {
			return script, nil
		}
	}
	return Script{}, fmt.Errorf("script %q not found", scriptName)
}

func executeScript(catalogDir, scriptPath string, args []string) error {
	cmd := exec.Command(scriptPath, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("JOY_CATALOG_DIR=%s", catalogDir))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
