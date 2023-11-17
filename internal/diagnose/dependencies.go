package diagnose

import (
	"github.com/nestoca/joy/internal/dependencies"
	"github.com/nestoca/joy/internal/style"
)

func diagnoseDependencies(requiredDependencies, optionalDependencies []*dependencies.Dependency, builder DiagnosticBuilder) {
	builder.StartDiagnostic("Dependencies")
	defer builder.EndDiagnostic()

	func() {
		builder.StartSection("Required dependencies")
		defer builder.EndSection()

		for _, dep := range requiredDependencies {
			if dep.IsInstalled() {
				builder.AddSuccess("%s installed", style.Code(dep.Command))
			} else {
				builder.AddError("%s missing (see %s)", style.Code(dep.Command), style.Link(dep.Url))
			}
		}
	}()

	func() {
		builder.StartSection("Optional dependencies")
		defer builder.EndSection()
		for _, dep := range optionalDependencies {
			if dep.IsInstalled() {
				builder.AddSuccess("%s installed", style.Code(dep.Command))
			} else {
				builder.AddInfo("%s missing (see %s) but only required by:", style.Code(dep.Command), style.Link(dep.Url))
				builder.StartSection("")
				for _, cmd := range dep.RequiredBy {
					builder.AddInfo(style.Code("joy " + cmd))
				}
				builder.EndSection()
			}
		}
	}()
}
