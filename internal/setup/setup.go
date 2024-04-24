package setup

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/nestoca/survey/v2"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/diagnostics"
	"github.com/nestoca/joy/internal/github"
	"github.com/nestoca/joy/internal/style"
)

const (
	defaultCatalogDir = "~/.joy"
	separator         = "————————————————————————————————————————————————————————————————————————————————"
)

func Setup(version, configDir, catalogDir, catalogRepo string) error {
	fmt.Println("👋 Hey there, let's kickstart your most joyful CD experience! ☀️")
	fmt.Println(separator)

	// Setup catalog and config
	fmt.Print("🛠️ Let's first set up your configuration and catalog repo...\n\n")
	catalogDir, err := setupCatalog(configDir, catalogDir, catalogRepo)
	if err != nil {
		return err
	}
	cfg, err := setupConfig(configDir, catalogDir)
	if err != nil {
		return err
	}

	fmt.Println(separator)

	// Run diagnostics
	fmt.Print("🔍 Let's run a few diagnostics to check everything is in order...\n\n")
	_, err = fmt.Println(diagnostics.OutputWithGlobalStats(diagnostics.Evaluate(version, cfg)))
	return err
}

func setupConfig(configDir string, catalogDir string) (*config.Config, error) {
	// Try loading config file from given or default location
	cfg, err := config.Load(configDir, catalogDir)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Save config file
	cfg.CatalogDir = catalogDir
	err = cfg.Save()
	if err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}
	fmt.Printf("✅ Saved config to file %s\n", style.Code(cfg.FilePath))
	return cfg, nil
}

func setupCatalog(configDir string, catalogDir string, catalogRepo string) (string, error) {
	var err error
	catalogDir, err = getCatalogDir(configDir, catalogDir)
	if err != nil {
		return "", err
	}

	// Check if catalog directory exists
	if _, err := os.Stat(catalogDir); err != nil {
		if os.IsNotExist(err) {
			err := cloneCatalog(catalogRepo, catalogDir)
			if err != nil {
				return "", err
			}
		} else {
			return "", fmt.Errorf("checking for catalog directory %s: %w", catalogDir, err)
		}
	}

	return catalogDir, nil
}

func getCatalogDir(configDir string, catalogDir string) (string, error) {
	if catalogDir == "" {
		// Try loading catalog dir from config file to use as prompt default value
		cfg, err := config.Load(configDir, catalogDir)
		if err == nil {
			catalogDir = cfg.CatalogDir
		} else {
			catalogDir = defaultCatalogDir
		}

		// Prompt user for catalog directory using survey (defaults to $HOME/.joy)
		err = survey.AskOne(&survey.Input{
			Message: "🎯 Where does (or should) your local catalog reside?",
			Help:    "This is where we will clone your catalog repo, but only if it's not already there.",
			Default: catalogDir,
		},
			&catalogDir,
			survey.WithValidator(survey.Required),
		)
		if err != nil {
			return "", fmt.Errorf("prompting for catalog directory: %w", err)
		}
	}

	// Expand tilde to home directory
	homePrefix := "~/"
	if strings.HasPrefix(catalogDir, homePrefix) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting home directory: %w", err)
		}
		catalogDir = path.Join(homeDir, strings.TrimPrefix(catalogDir, homePrefix))
	}

	return filepath.Abs(catalogDir)
}

func cloneCatalog(catalogRepo, catalogDir string) error {
	shouldClone := true
	if catalogRepo == "" {
		if err := survey.AskOne(
			&survey.Confirm{
				Message: "🤷 No trace of catalog at given location, clone it?",
				Default: true,
			},
			&shouldClone,
		); err != nil {
			return fmt.Errorf("prompting for catalog cloning: %w", err)
		}

		if !shouldClone {
			return fmt.Errorf("😬 Sorry, cannot continue without catalog!")
		}

		if err := survey.AskOne(
			&survey.Input{Message: "📦 What's your catalog repo URL (format: org/repo)?"},
			&catalogRepo,
			survey.WithValidator(survey.Required),
		); err != nil {
			return fmt.Errorf("prompting for catalog repo: %w", err)
		}
	}

	fmt.Println()

	opts := github.CloneOptions{
		Repo:   catalogRepo,
		OutDir: catalogDir,
	}

	if err := github.CloneInteractive(".", opts); err != nil {
		return fmt.Errorf("cloning catalog: %w", err)
	}

	fmt.Println()
	fmt.Printf("✅ Cloned catalog from %s to %s\n", style.Link(catalogRepo), style.Code(catalogDir))

	return nil
}
