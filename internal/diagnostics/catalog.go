package diagnostics

import (
	"fmt"
	"io/fs"
	"os"
	"reflect"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/git"
	"github.com/nestoca/joy/internal/references"
	"github.com/nestoca/joy/internal/style"
	"github.com/nestoca/joy/pkg/catalog"
)

type GitOpts struct {
	IsValid               func(string) bool
	GetUncommittedChanges func(string) ([]string, error)
	GetCurrentBranch      func(string) (string, error)
	IsInSyncWithRemote    func(string, string) (bool, error)
	GetCurrentCommit      func(string) (string, error)
}

type CatalogOpts struct {
	Stat         func(string) (fs.FileInfo, error)
	CheckCatalog func(string) error
	Git          GitOpts
}

func diagnoseCatalog(catalogDir string, cat *catalog.Catalog, opts CatalogOpts) (group Group) {
	if opts.Stat == nil {
		opts.Stat = os.Stat
	}
	if opts.CheckCatalog == nil {
		opts.CheckCatalog = config.CheckCatalogDir
	}

	if reflect.ValueOf(opts.Git).IsZero() {
		opts.Git = GitOpts{
			IsValid:               git.IsValid,
			GetUncommittedChanges: git.GetUncommittedChanges,
			GetCurrentBranch:      git.GetCurrentBranch,
			IsInSyncWithRemote:    git.IsBranchInSyncWithRemote,
			GetCurrentCommit:      git.GetCurrentCommit,
		}
	}

	group.Title = "Catalog"
	group.topLevel = true

	group.SubGroups = append(group.SubGroups, func() (group Group) {
		group.Title = "Git working copy"

		group.AddMsg(success, "Working copy is valid")

		uncommittedChanges, err := opts.Git.GetUncommittedChanges(catalogDir)
		if err != nil {
			group.AddMsg(failed, label("Failed checking for uncommitted changes", err.Error()))
		}
		if len(uncommittedChanges) > 0 {
			group.AddMsg(
				warning,
				fmt.Sprintf("Working copy has %d uncommitted change(s)", len(uncommittedChanges)),
				msg(hint, fmt.Sprintf("Commit your changes or discard them using: %s", style.Code("joy git reset --hard && joy git clean -fd"))),
			)
		} else {
			group.AddMsg(success, "Working copy has no uncommitted changes")
		}

		const defaultBranch = "master"
		currentBranch, err := opts.Git.GetCurrentBranch(catalogDir)
		if err != nil {
			group.AddMsg(failed, label("Failed getting current branch", err.Error()))
		} else {
			if currentBranch != defaultBranch {
				group.AddMsg(
					warning,
					fmt.Sprintf("Default branch %s should be checked out (instead of %s)", style.Code(defaultBranch), style.Code(currentBranch)),
					msg(hint, fmt.Sprintf("Switch to default branch using: %s", style.Code("joy git checkout "+defaultBranch))),
				)
			} else {
				group.AddMsg(success, fmt.Sprintf("Default branch %s is checked out", style.Code(currentBranch)))
			}
		}

		isInSync, err := opts.Git.IsInSyncWithRemote(catalogDir, defaultBranch)
		if err != nil {
			group.AddMsg(failed, label("Failed checking default branch sync state", err.Error()))
		} else {
			if isInSync {
				group.AddMsg(success, "Default branch is in sync with remote")
			} else {
				group.AddMsg(
					warning,
					"Default branch is not in sync with remote",
					msg(hint, fmt.Sprintf("Update your working copy using: %s", style.Code("joy pull"))),
				)
			}
		}

		commit, err := opts.Git.GetCurrentCommit(catalogDir)
		if err != nil {
			group.AddMsg(failed, label("Failed getting current commit", err.Error()))
		} else {
			group.AddMsg(info, label("Current commit", commit))
		}

		return
	}())

	group.AddSubGroup(func() (group Group) {
		group.Title = "Resources"
		group.
			AddMsg(info, label("Environments", len(cat.Environments))).
			AddMsg(info, label("Projects", len(cat.Projects))).
			AddMsg(info, label("Releases", len(cat.Releases.Items)))
		return
	}())

	group.AddSubGroup(func() (group Group) {
		group.Title = "Cross-references"
		if err := cat.ResolveRefs(); err != nil {
			for _, err := range references.AsMissingErrors(err) {
				group.AddMsg(failed, err.StyledError())
			}
			return
		}
		group.AddMsg(success, "All resource cross-references resolved successfully")
		return
	}())

	return
}
