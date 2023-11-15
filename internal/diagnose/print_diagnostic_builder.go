package diagnose

import (
	"fmt"
	"strings"

	"github.com/nestoca/joy/internal/style"
)

func NewPrintDiagnosticBuilder() *PrintDiagnosticBuilder {
	return &PrintDiagnosticBuilder{}
}

type PrintDiagnosticBuilder struct {
	builder                        strings.Builder
	currentDiagnosticName          string
	currentDiagnosticStringBuilder *strings.Builder
	currentDiagnosticErrorCount    int
	currentDiagnosticWarningCount  int
	depth                          int
	errorCount                     int
	warningCount                   int
}

func (b *PrintDiagnosticBuilder) printItem(format string, a ...any) {
	// Use currentDiagnosticStringBuilder if we're in a diagnostic in order
	// to delay writing its header until we know whether it has any errors.
	builder := func() *strings.Builder {
		if b.currentDiagnosticStringBuilder != nil {
			return b.currentDiagnosticStringBuilder
		}
		return &b.builder
	}()

	for i := 0; i < b.depth; i++ {
		builder.WriteString("  ")
	}
	message := fmt.Sprintf(format, a...)
	builder.WriteString(message + "\n")
}

func (b *PrintDiagnosticBuilder) Start() {
	if b.depth != 0 {
		panic("Start called within a diagnostic")
	}
}

func (b *PrintDiagnosticBuilder) End() {
	if b.depth != 0 {
		panic("End called within a diagnostic")
	}
	if b.errorCount > 0 || b.warningCount > 0 {
		b.printItem("🚨 Diagnostics completed with %d error(s) and %d warning(s)", b.errorCount, b.warningCount)
	} else {
		b.printItem("🚀 All systems nominal. Houston, we're cleared for launch!")
	}
}

func (b *PrintDiagnosticBuilder) StartDiagnostic(name string) {
	if b.depth != 0 {
		panic("StartDiagnostic called within another diagnostic")
	}
	b.currentDiagnosticName = name
	b.currentDiagnosticStringBuilder = &strings.Builder{}
	b.depth = 1
}

func (b *PrintDiagnosticBuilder) EndDiagnostic() {
	if b.depth == 0 {
		panic("EndDiagnostic called outside a diagnostic")
	}
	if b.depth > 1 {
		panic("EndDiagnostic called within a section")
	}

	emoji := func() string {
		if b.currentDiagnosticErrorCount > 0 {
			return "💔"
		}
		if b.currentDiagnosticWarningCount > 0 {
			return "⚠️"
		}
		return "✅"
	}()
	b.builder.WriteString(fmt.Sprintf("%s %s\n", emoji, style.DiagnosticHeader(b.currentDiagnosticName)))
	b.builder.WriteString(b.currentDiagnosticStringBuilder.String())
	b.builder.WriteString("\n")

	b.currentDiagnosticStringBuilder = nil
	b.currentDiagnosticErrorCount = 0
	b.currentDiagnosticWarningCount = 0
	b.depth = 0
}

func (b *PrintDiagnosticBuilder) StartSection(name string) {
	if b.depth == 0 {
		panic("StartSection called outside a diagnostic")
	}
	if name != "" {
		b.printItem(style.DiagnosticSection(name))
	}
	b.depth++
}

func (b *PrintDiagnosticBuilder) EndSection() {
	if b.depth == 0 {
		panic("EndSection called outside a diagnostic")
	}
	if b.depth == 1 {
		panic("EndSection called outside a section")
	}
	b.depth--
}

func (b *PrintDiagnosticBuilder) AddInfo(format string, a ...any) {
	if b.depth == 0 {
		panic("AddInfo called outside a diagnostic")
	}
	message := fmt.Sprintf(format, a...)
	b.printItem("➡️ %s", message)
}

func (b *PrintDiagnosticBuilder) AddSuccess(format string, a ...any) {
	if b.depth == 0 {
		panic("AddSuccess called outside a diagnostic")
	}
	message := fmt.Sprintf(format, a...)
	b.printItem("✅ %s", message)
}

func (b *PrintDiagnosticBuilder) AddWarning(format string, a ...any) {
	if b.depth == 0 {
		panic("AddWarning called outside a diagnostic")
	}
	message := fmt.Sprintf(format, a...)
	b.printItem("⚠️ %s", message)
	b.currentDiagnosticWarningCount++
	b.warningCount++
}

func (b *PrintDiagnosticBuilder) AddError(format string, a ...any) {
	if b.depth == 0 {
		panic("AddError called outside a diagnostic")
	}
	message := fmt.Sprintf(format, a...)
	b.printItem("💔 %s", message)
	b.currentDiagnosticErrorCount++
	b.errorCount++
}

func (b *PrintDiagnosticBuilder) AddRecommendation(format string, a ...any) {
	if b.depth == 0 {
		panic("AddRecommendation called outside a diagnostic")
	}
	message := fmt.Sprintf(format, a...)
	b.printItem("   👉 %s", message)
}

func (b *PrintDiagnosticBuilder) String() string {
	return b.builder.String()
}
