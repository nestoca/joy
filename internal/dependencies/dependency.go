package dependencies

import (
	"fmt"
	"github.com/nestoca/joy/internal/style"
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

func (d *Dependency) Check() error {
	if !d.IsInstalled() {
		fmt.Printf("😅 Oops! This command requires %s dependency (see: %s)\n", style.Code(d.Command), style.Link(d.Url))
		return fmt.Errorf("missing %s dependency", d.Command)
	}
	return nil
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

func CheckAllRequired() error {
	var allErrors []error
	for _, dep := range AllRequired {
		if dep.IsRequired {
			err := dep.Check()
			if err != nil {
				allErrors = append(allErrors, err)
			}
		}
	}
	if len(allErrors) > 0 {
		return fmt.Errorf("missing required dependencies: %v", allErrors)
	}
	return nil
}
