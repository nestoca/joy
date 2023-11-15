package diagnose

import (
	"os"

	"github.com/nestoca/joy/internal/style"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/references"

	"github.com/nestoca/joy/pkg/catalog"
)

func diagnoseCatalog(catalogDir string, builder DiagnosticBuilder) {
	builder.StartDiagnostic("Catalog")
	defer builder.EndDiagnostic()

	// Diagnose catalog working copy
	func() {
		builder.StartSection("Git working copy")
		defer builder.EndSection()

		if _, err := os.Stat(catalogDir); os.IsNotExist(err) {
			AddLabelAndError(builder, "Directory does not exist", catalogDir)
			return
		} else {
			AddLabelAndSuccess(builder, "Directory exists", catalogDir)
		}

		if git.IsValid(catalogDir) {
			builder.AddSuccess("Working copy is valid")
		} else {
			builder.AddError("Working copy is invalid")
			return
		}

		uncommittedChanges, err := git.GetUncommittedChanges(catalogDir)
		if err != nil {
			AddLabelAndError(builder, "Failed checking for uncommitted changes", "%s", err)
		}
		if len(uncommittedChanges) > 0 {
			builder.AddWarning("Working copy has %d uncommitted change(s)", len(uncommittedChanges))
			builder.AddRecommendation("Commit your changes or discard them using: %s", style.Code("joy git reset --hard && joy git clean -fd"))
		} else {
			builder.AddSuccess("Working copy has no uncommitted changes")
		}

		defaultBranch, err := git.GetDefaultBranch(catalogDir)
		if err != nil {
			AddLabelAndError(builder, "Failed getting default branch", "%s", err)
		}
		currentBranch, err := git.GetCurrentBranch(catalogDir)
		if err != nil {
			AddLabelAndError(builder, "Failed getting current branch", "%s", err)
		} else {
			if currentBranch != defaultBranch {
				builder.AddWarning("Default branch %s should be checked out (instead of %s)", style.Code(defaultBranch), style.Code(currentBranch))
				builder.AddRecommendation("Switch to default branch using: %s", style.Code("joy git checkout "+defaultBranch))
			} else {
				builder.AddSuccess("Default branch %s is checked out", style.Code(currentBranch))
			}
		}

		isInSync, err := git.IsBranchInSyncWithRemote(catalogDir, defaultBranch)
		if err != nil {
			AddLabelAndError(builder, "Failed checking default branch sync state", "%s", err)
		} else {
			if isInSync {
				builder.AddSuccess("Default branch is in sync with remote")
			} else {
				builder.AddWarning("Default branch is not in sync with remote")
				builder.AddRecommendation("Update your working copy using: %s", style.Code("joy pull"))
			}
		}

		commit, err := git.GetCurrentCommit(catalogDir)
		if err != nil {
			AddLabelAndError(builder, "Failed getting current commit", "%s", err)
		} else {
			AddLabelAndInfo(builder, "Current commit", "%s", commit)
		}
	}()

	// Load catalog
	cat := func() *catalog.Catalog {
		builder.StartSection("Loading catalog")
		defer builder.EndSection()

		// Check catalog dir
		err := config.CheckCatalogDir(catalogDir)
		if err != nil {
			AddLabelAndError(builder, "Catalog not detected", "%s", err)
			return nil
		} else {
			builder.AddSuccess("Catalog detected")
		}

		opts := catalog.LoadOpts{
			Dir:          catalogDir,
			LoadEnvs:     true,
			LoadProjects: true,
			LoadReleases: true,
			ResolveRefs:  false,
		}
		cat, err := catalog.Load(opts)
		if err != nil {
			AddLabelAndError(builder, "Failed loading catalog", "%s", err)
		}
		builder.AddSuccess("Catalog loaded successfully")

		return cat
	}()
	if cat == nil {
		return
	}

	// Print catalog stats
	func() {
		builder.StartSection("Resources")
		defer builder.EndSection()

		AddLabelAndInfo(builder, "Environments", "%d", len(cat.Environments))
		AddLabelAndInfo(builder, "Projects", "%d", len(cat.Projects))
		AddLabelAndInfo(builder, "Releases", "%d", len(cat.Releases.Items))
	}()

	// Resolve catalog references, checking for missing references
	func() {
		builder.StartSection("Cross-references")
		defer builder.EndSection()

		err := cat.ResolveRefs()
		if err != nil {
			errs := references.AsMissingErrors(err)
			for _, err := range errs {
				builder.AddError(err.StyledError())
			}
		} else {
			builder.AddSuccess("All resource cross-references resolved successfully")
		}
	}()
}
