package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/AlecAivazis/survey/v2"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/dependencies"
	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/pkg/catalog"
)

const (
	defaultCatalogDir = "~/.joy"
	separator         = "————————————————————————————————————————————————————————————————————————————————"
)

func Setup(configDir, catalogDir, catalogRepo string) error {
	fmt.Println("👋 Hey there, let's kickstart your most joyful CD experience! ☀️")
	fmt.Println(separator)

	// Setup catalog and config
	fmt.Print("🛠️ Let's first set up your configuration and catalog repo...\n\n")
	catalogDir, err := setupCatalog(configDir, catalogDir, catalogRepo)
	if err != nil {
		return err
	}
	err = setupConfig(configDir, catalogDir)
	if err != nil {
		return err
	}
	fmt.Println(separator)

	// Check dependencies
	fmt.Print("🧐 Hmm, let's now see what dependencies you've got humming under the hood...\n\n")
	if err := checkDependencies(); err != nil {
		return err
	}
	fmt.Println(separator)

	fmt.Println("🚀 All systems nominal. Houston, we're cleared for launch!")
	return nil
}

func setupConfig(configDir string, catalogDir string) error {
	// Try loading config file from given or default location
	cfg, err := config.Load(configDir, catalogDir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Save config file
	cfg.CatalogDir = catalogDir
	err = cfg.Save()
	if err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Printf("✅ Saved config to file %s\n", style.Code(cfg.FilePath))
	return nil
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

	cat, err := loadCatalog(catalogDir)
	if err != nil {
		return "", err
	}

	printCatalogSummary(cat)
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
	return catalogDir, nil
}

func loadCatalog(catalogDir string) (*catalog.Catalog, error) {
	cat, err := catalog.Load(catalog.LoadOpts{
		Dir:          catalogDir,
		LoadEnvs:     true,
		LoadProjects: true,
		LoadReleases: true,
		ResolveRefs:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("🤯 Whoa! Found the catalog, but failed to load it. Check this error and try again:\n%v", err)
	}
	return cat, nil
}

func cloneCatalog(catalogRepo, catalogDir string) error {
	shouldClone := true
	if catalogRepo == "" {
		// Prompt user whether to clone it
		err := survey.AskOne(&survey.Confirm{
			Message: "🤷 No trace of catalog at given location, clone it?",
			Default: true,
		},
			&shouldClone,
		)
		if err != nil {
			return fmt.Errorf("prompting for catalog cloning: %w", err)
		}

		// Prompt user for catalog repo
		err = survey.AskOne(&survey.Input{
			Message: "📦 What's your catalog repo URL?",
		},
			&catalogRepo,
			survey.WithValidator(survey.Required),
		)
		if err != nil {
			return fmt.Errorf("prompting for catalog repo: %w", err)
		}
	}

	// Clone catalog
	if shouldClone {
		cmd := exec.Command("git", "clone", catalogRepo, catalogDir)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("cloning catalog: %s", output)
		}
		fmt.Printf("✅ Cloned catalog from %s to %s\n", style.Link(catalogRepo), style.Code(catalogDir))
	} else {
		return fmt.Errorf("😬 Sorry, cannot continue without catalog!")
	}
	return nil
}

func printCatalogSummary(cat *catalog.Catalog) {
	envCount := len(cat.Environments)
	projectCount := len(cat.Projects)
	releaseCount := len(cat.Releases.Items)
	if envCount == 0 || projectCount == 0 || releaseCount == 0 {
		fmt.Print("🦗 Crickets... No ")
		if envCount == 0 {
			fmt.Print("environments")
		}
		if projectCount == 0 {
			if envCount == 0 {
				fmt.Print("/")
			}
			fmt.Print("projects")
		}
		if releaseCount == 0 {
			if envCount == 0 || projectCount == 0 {
				fmt.Print("/")
			}
			fmt.Print("releases")
		}
		fmt.Println(" found. Please add some to the catalog.")
	} else {
		fmt.Printf("✅ Catalog loaded! You've got %d environments, %d projects, and %d releases. Nice! 👍\n", envCount, projectCount, releaseCount)
	}
}

func checkDependencies() error {
	missingRequired := false
	for _, dep := range dependencies.AllRequired {
		if dep.IsInstalled() {
			fmt.Printf("✅ Found %s required dependency.\n", style.Code(dep.Command))
		} else {
			fmt.Printf("❌ The %s required dependency is missing (see %s).\n", style.Code(dep.Command), style.Link(dep.Url))
			missingRequired = true
		}
	}
	for _, dep := range dependencies.AllOptional {
		if dep.IsInstalled() {
			fmt.Printf("✅ Found %s optional dependency.\n", style.Code(dep.Command))
		} else {
			fmt.Printf(fmt.Sprintf("🤷 The %s optional dependency is missing (see: %s) but only required by those commands:\n", style.Code(dep.Command), style.Link(dep.Url)))
			for _, cmd := range dep.RequiredBy {
				fmt.Printf(" 🔹 %s\n", style.Code("joy "+cmd))
			}
		}
	}

	if missingRequired {
		fmt.Println()
		return fmt.Errorf("😅 Oops! Joy requires those dependencies to operate. Please install them and try again! 🙏")
	}

	return nil
}
