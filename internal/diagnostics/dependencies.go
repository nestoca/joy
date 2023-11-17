package diagnostics

import (
	"fmt"

	"github.com/nestoca/joy/internal/dependencies"
	"github.com/nestoca/joy/internal/style"
)

func diagnoseDependencies() (group Group) {
	group.Title = "Dependencies"
	group.toplevel = true

	group.AddSubGroup(func() (required Group) {
		required.Title = "Required dependencies"
		for _, dep := range dependencies.AllRequired {
			if !dep.IsInstalled() {
				required.AddMsg(failed, fmt.Sprintf("%s missing (see %s)", style.Code(dep.Command), style.Link(dep.Url)))
				continue
			}
			required.AddMsg(success, fmt.Sprintf("%s installed", style.Code(dep.Command)))
		}
		return
	}())

	group.AddSubGroup(func() (optional Group) {
		optional.Title = "Optional dependencies"
		for _, dep := range dependencies.AllOptional {
			if !dep.IsInstalled() {
				optional.AddMsg(
					failed,
					fmt.Sprintf("%s missing (see %s) but only required by:", style.Code(dep.Command), style.Link(dep.Url)),
					func() (msgs Messages) {
						for _, cmd := range dep.RequiredBy {
							msgs = append(msgs, msg(info, style.Code("joy "+cmd)))
						}
						return
					}()...,
				)
				continue
			}
			optional.AddMsg(success, fmt.Sprintf("%s installed", style.Code(dep.Command)))
		}
		return
	}())

	return
}
