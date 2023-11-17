//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE
package diagnose

import (
	"fmt"

	"github.com/nestoca/joy/internal/style"
)

type DiagnosticBuilder interface {
	Start()
	End()

	StartDiagnostic(name string)
	EndDiagnostic()

	StartSection(name string)
	EndSection()

	AddInfo(format string, a ...any)
	AddSuccess(format string, a ...any)
	AddWarning(format string, a ...any)
	AddError(format string, a ...any)
	AddRecommendation(format string, a ...any)
}

func AddLabelAndInfo(builder DiagnosticBuilder, label, format string, a ...any) {
	message := fmt.Sprintf(style.DiagnosticLabel(label+": ")+format, a...)
	builder.AddInfo(message)
}

func AddLabelAndSuccess(builder DiagnosticBuilder, label, format string, a ...any) {
	message := fmt.Sprintf(style.DiagnosticLabel(label+": ")+format, a...)
	builder.AddSuccess(message)
}

func AddLabelAndWarning(builder DiagnosticBuilder, label, format string, a ...any) {
	message := fmt.Sprintf(style.DiagnosticLabel(label+": ")+format, a...)
	builder.AddWarning(message)
}

func AddLabelAndError(builder DiagnosticBuilder, label, format string, a ...any) {
	message := fmt.Sprintf(style.DiagnosticLabel(label+": ")+format, a...)
	builder.AddError(message)
}
