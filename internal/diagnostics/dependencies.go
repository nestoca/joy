package diagnostics

import (
	"fmt"

	"github.com/nestoca/joy/internal/dependencies"
	"github.com/nestoca/joy/internal/style"
)

func diagnoseDependencies(required, optional []*dependencies.Dependency) (group Group) {
	group.Title = "Dependencies"
	group.topLevel = true

	group.AddSubGroup(func() (group Group) {
		group.Title = "Required dependencies"
		for _, dep := range required {
			if !dep.IsInstalled() {
				group.AddMsg(failed, fmt.Sprintf("%s missing (see %s)", style.Code(dep.Command), style.Link(dep.Url)))
				continue
			}
			group.AddMsg(success, fmt.Sprintf("%s installed", style.Code(dep.Command)))
		}
		return
	}())

	group.AddSubGroup(func() (group Group) {
		group.Title = "Optional dependencies"
		for _, dep := range optional {
			if !dep.IsInstalled() {
				group.AddMsg(
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
			group.AddMsg(success, fmt.Sprintf("%s installed", style.Code(dep.Command)))
		}
		return
	}())

	return
}
