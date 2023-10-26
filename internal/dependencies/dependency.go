package dependencies

import (
	"fmt"
	"github.com/nestoca/joy/internal/style"
	"os"
	"os/exec"
)

type Dependency struct {
	// Command that should be found in PATH
	Command string

	// Url to dependency's website
	Url string

	// IsRequired indicates whether this is a core dependency required to run joy
	IsRequired bool

	// RequiredBy lists which joy sub-commands require this dependency
	RequiredBy []string
}

func (d *Dependency) IsInstalled() bool {
	cmd := exec.Command("command", "-v", d.Command)
	return cmd.Run() == nil
}

func (d *Dependency) MustBeInstalled() {
	if !d.IsInstalled() {
		fmt.Printf("😅 Oops! This command requires %s dependency (see: %s)\n", style.Code(d.Command), style.Link(d.Url))
		os.Exit(1)
	}
}

var AllRequired []*Dependency
var AllOptional []*Dependency

func Add(dep *Dependency) {
	if dep.IsRequired {
		AllRequired = append(AllRequired, dep)
	} else {
		AllOptional = append(AllOptional, dep)
	}
}

func AllRequiredMustBeInstalled() {
	missingRequired := false
	for _, dep := range AllRequired {
		if dep.IsRequired && !dep.IsInstalled() {
			fmt.Printf("❌ The %s required dependency is missing (see %s).\n", style.Code(dep.Command), style.Link(dep.Url))
			missingRequired = true
		}
	}
	if missingRequired {
		fmt.Println("😅 Oops! Joy requires those dependencies to operate. Please install them and try again! 🙏")
		os.Exit(1)
	}
}
