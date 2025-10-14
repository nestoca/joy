package validate

import (
	"context"
	"errors"
	"fmt"

	"github.com/davidmdm/x/xerr"
	"golang.org/x/mod/semver"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal/release/render"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/helm"
)

type ValidateParams struct {
	Releases   []*v1alpha1.Release
	Helm       helm.PullRenderer
	ChartCache helm.ChartCache
}

func Validate(ctx context.Context, params ValidateParams) error {
	var errs []error
	for _, release := range params.Releases {
		chart, err := params.ChartCache.GetReleaseChartFS(ctx, release)
		if err != nil {
			return fmt.Errorf("getting release chart: %w", err)
		}

		validateParams := ValidateReleaseParams{
			Chart:   chart,
			Release: release,
			Helm:    params.Helm,
		}

		if err := ValidateRelease(ctx, validateParams); err != nil {
			errs = append(errs, fmt.Errorf("%s/%s: %w", release.Name, release.Environment.Name, err))
		}
	}

	return xerr.MultiErrOrderedFrom("validating releases", errs...)
}

type ValidateReleaseParams struct {
	Release *v1alpha1.Release
	Chart   *helm.ChartFS
	Helm    helm.PullRenderer
}

func ValidateRelease(ctx context.Context, params ValidateReleaseParams) error {
	if !params.Release.Environment.Spec.Promotion.FromPullRequests && !params.Release.Project.Spec.SkipPreReleaseCheck {
		version := "v" + params.Release.Spec.Version
		if semver.Prerelease(version)+semver.Build(version) != "" {
			return fmt.Errorf("invalid version: prerelease branches not allowed: %s", params.Release.Spec.Version)
		}
	}

	if yml.HasLockedTodos(params.Release.File.Tree) {
		return errors.New("contains locked TODO")
	}

	renderOpts := render.RenderParams{
		Release: params.Release,
		Chart:   params.Chart,
		Helm:    params.Helm,
	}

	if _, err := render.Render(ctx, renderOpts); err != nil {
		return err
	}

	return nil
}
