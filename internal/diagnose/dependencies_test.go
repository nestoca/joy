package diagnose

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/nestoca/joy/internal/dependencies"
)

func TestDependencies(t *testing.T) {
	requiredBy := []string{"cmd1", "cmd2"}
	installedDep1 := &dependencies.Dependency{
		Command:    "bash",
		Url:        "https://example.com/bash",
		RequiredBy: requiredBy,
	}
	installedDep2 := &dependencies.Dependency{
		Command:    "sh",
		Url:        "https://example.com/sh",
		RequiredBy: requiredBy,
	}
	missingDep1 := &dependencies.Dependency{
		Command:    "missing-dependency",
		Url:        "https://example.com/missing",
		RequiredBy: requiredBy,
	}

	testCases := []struct {
		name                 string
		requiredDependencies []*dependencies.Dependency
		optionalDependencies []*dependencies.Dependency
		setupFunc            func(builder *MockDiagnosticBuilder)
	}{
		{
			name:                 "no dependencies",
			requiredDependencies: nil,
			optionalDependencies: nil,
			setupFunc: func(builder *MockDiagnosticBuilder) {
				builder.EXPECT().StartDiagnostic("Dependencies")

				builder.EXPECT().StartSection("Required dependencies")
				builder.EXPECT().EndSection()

				builder.EXPECT().StartSection("Optional dependencies")
				builder.EXPECT().EndSection()

				builder.EXPECT().EndDiagnostic()
			},
		},
		{
			name:                 "all required and optional dependencies installed",
			requiredDependencies: []*dependencies.Dependency{installedDep1, installedDep2},
			optionalDependencies: []*dependencies.Dependency{installedDep1, installedDep2},
			setupFunc: func(builder *MockDiagnosticBuilder) {
				builder.EXPECT().StartDiagnostic("Dependencies")

				builder.EXPECT().StartSection("Required dependencies")
				builder.EXPECT().AddSuccess(gomock.Any(), gomock.Any())
				builder.EXPECT().AddSuccess(gomock.Any(), gomock.Any())
				builder.EXPECT().EndSection()

				builder.EXPECT().StartSection("Optional dependencies")
				builder.EXPECT().AddSuccess(gomock.Any(), gomock.Any())
				builder.EXPECT().AddSuccess(gomock.Any(), gomock.Any())
				builder.EXPECT().EndSection()

				builder.EXPECT().EndDiagnostic()
			},
		},
		{
			name:                 "missing required dependency",
			requiredDependencies: []*dependencies.Dependency{installedDep1, missingDep1},
			optionalDependencies: []*dependencies.Dependency{installedDep1, installedDep2},
			setupFunc: func(builder *MockDiagnosticBuilder) {
				builder.EXPECT().StartDiagnostic("Dependencies")

				builder.EXPECT().StartSection("Required dependencies")
				builder.EXPECT().AddSuccess(gomock.Any(), gomock.Any())
				builder.EXPECT().AddError(gomock.Any(), gomock.Any())
				builder.EXPECT().EndSection()

				builder.EXPECT().StartSection("Optional dependencies")
				builder.EXPECT().AddSuccess(gomock.Any(), gomock.Any())
				builder.EXPECT().AddSuccess(gomock.Any(), gomock.Any())
				builder.EXPECT().EndSection()

				builder.EXPECT().EndDiagnostic()
			},
		},
		{
			name:                 "missing optional dependency",
			requiredDependencies: []*dependencies.Dependency{installedDep1, installedDep2},
			optionalDependencies: []*dependencies.Dependency{installedDep1, missingDep1},
			setupFunc: func(builder *MockDiagnosticBuilder) {
				builder.EXPECT().StartDiagnostic("Dependencies")

				builder.EXPECT().StartSection("Required dependencies")
				builder.EXPECT().AddSuccess(gomock.Any(), gomock.Any())
				builder.EXPECT().AddSuccess(gomock.Any(), gomock.Any())
				builder.EXPECT().EndSection()

				builder.EXPECT().StartSection("Optional dependencies")
				builder.EXPECT().AddSuccess(gomock.Any(), gomock.Any())
				builder.EXPECT().AddInfo(gomock.Any(), gomock.Any())

				builder.EXPECT().StartSection("")
				builder.EXPECT().AddInfo(gomock.Any())
				builder.EXPECT().AddInfo(gomock.Any())
				builder.EXPECT().EndSection()

				builder.EXPECT().EndSection()

				builder.EXPECT().EndDiagnostic()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mocks
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			builder := NewMockDiagnosticBuilder(ctrl)

			// Setup
			tc.setupFunc(builder)

			// Run test
			diagnoseDependencies(tc.requiredDependencies, tc.optionalDependencies, builder)
		})
	}
}
