package setup

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/dependencies"
	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/pkg/catalog"
	"os"
	"os/exec"
	"path"
	"strings"
)

const defaultCatalogDir = "~/.joy"

func Setup(configDir, catalogDir, catalogRepo string) error {
	fmt.Println("👋 Hey there, let's kickstart your most joyful CD experience! ☀️")
	separator := "————————————————————————————————————————————————————————————————————————————————"
	fmt.Println(separator)
	fmt.Print("🛠️ Let's first set up your configuration and catalog repo...\n\n")
	// Determine catalog directory
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
			return fmt.Errorf("prompting for catalog directory: %w", err)
		}
	}

	// Expand tilde to home directory
	homePrefix := "~/"
	if strings.HasPrefix(catalogDir, homePrefix) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}
		catalogDir = path.Join(homeDir, strings.TrimPrefix(catalogDir, homePrefix))
	}

	// Check if catalog directory exists
	if _, err := os.Stat(catalogDir); err != nil {
		if os.IsNotExist(err) {
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
				fmt.Println("😬 Sorry, cannot continue without catalog!")
				os.Exit(1)
			}

		} else {
			return fmt.Errorf("checking for catalog directory %s: %w", catalogDir, err)
		}
	}

	// Check catalog directory
	cat, err := catalog.Load(catalog.LoadOpts{
		Dir:          catalogDir,
		LoadEnvs:     true,
		LoadProjects: true,
		LoadReleases: true,
		ResolveRefs:  true,
	})
	if err != nil {
		fmt.Printf("🤯 Whoa! Found the catalog, but it's speaking gibberish. Check this error and try again:\n%v\n", err)
		os.Exit(1)
	}

	// Print catalog content summary
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
		fmt.Printf("🤩 Wowza! Your catalog's bursting with %d environments, %d projects, and %d releases.\n", envCount, projectCount, releaseCount)
	}

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
	fmt.Printf("✅ Saved config file to %s\n", style.Code(cfg.FilePath))

	// Check for required and optional dependencies
	fmt.Println(separator)
	fmt.Print("🧐 Hmm, let's now see what dependencies you've got humming under the hood...\n\n")
	missingRequired := false
	for _, dep := range dependencies.AllRequired {
		if dep.IsInstalled() {
			fmt.Printf("✅ Found required %s dependency.\n", style.Code(dep.Command))
		} else {
			fmt.Printf("❌ Missing required %s dependency.\n", style.Code(dep.Command))
			missingRequired = true
		}
	}
	missingOptional := false
	for _, dep := range dependencies.AllOptional {
		if dep.IsInstalled() {
			fmt.Printf("✅ Found optional %s dependency.\n", style.Code(dep.Command))
		} else {
			fmt.Printf(fmt.Sprintf("🤷 The optional %s dependency is missing (see: %s) but only required by those commands:\n", style.Code(dep.Command), style.Link(dep.Url)))
			for _, cmd := range dep.RequiredBy {
				fmt.Printf(" 🔹 %s\n", style.Code("joy "+cmd))
			}
			missingOptional = true
		}
	}

	// Print dependency summary
	fmt.Println()
	if missingRequired {
		fmt.Printf("😬 Yikes! Without all required dependencies, %s is more like %s. Install 'em and let's get joyful!\n", style.Code("joy"), style.Code("oy"))
		os.Exit(1)
	} else if missingOptional {
		fmt.Println("🍒 Cherry on top? Collect all optional dependencies for the sweetest experience!")
	} else {
		fmt.Println("🎉 Woohoo! You've got the whole shebang for maximum joy!")
	}

	fmt.Println(separator)
	fmt.Println("🚀 All systems nominal. Houston, we're cleared for launch!")
	return nil
}
