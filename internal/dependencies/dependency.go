package dependencies

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/nestoca/joy/internal/style"
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
	return exec.Command("/bin/bash", "-c", "command -v "+d.Command).Run() == nil
}

func (d *Dependency) MustBeInstalled() error {
	if !d.IsInstalled() {
		return fmt.Errorf("😅 Oops! This command requires %s dependency (see: %s)", style.Code(d.Command), style.Link(d.Url))
	}
	return nil
}

var (
	AllRequired []*Dependency
	AllOptional []*Dependency
)

func Add(dep *Dependency) {
	if dep.IsRequired {
		AllRequired = append(AllRequired, dep)
	} else {
		AllOptional = append(AllOptional, dep)
	}
}

func AllRequiredMustBeInstalled(out io.Writer) error {
	missingRequired := false
	for _, dep := range AllRequired {
		if dep.IsRequired && !dep.IsInstalled() {
			_, _ = fmt.Fprintf(out, "❌ The %s required dependency is missing (see %s).\n", style.Code(dep.Command), style.Link(dep.Url))
			missingRequired = true
		}
	}
	if missingRequired {
		return fmt.Errorf("😅 Oops! Joy requires those dependencies to operate. Please install them and try again! 🙏")
	}
	return nil
}
