package validate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/davidmdm/x/xerr"
	"golang.org/x/mod/semver"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/release/render"
	"github.com/nestoca/joy/internal/yml"
	"github.com/nestoca/joy/pkg/helm"
)

type ValidateParams struct {
	Releases     []*v1alpha1.Release
	ValueMapping *config.ValueMapping
	Helm         helm.PullRenderer
	ChartCache   helm.ChartCache
}

func Validate(ctx context.Context, params ValidateParams) error {
	var errs []error
	for _, release := range params.Releases {
		chart, err := params.ChartCache.GetReleaseChartFS(ctx, release)
		if err != nil {
			return fmt.Errorf("getting release chart: %w", err)
		}

		validateParams := ValidateReleaseParams{
			Chart:        chart,
			Release:      release,
			ValueMapping: params.ValueMapping,
			Helm:         params.Helm,
		}

		if err := ValidateRelease(ctx, validateParams); err != nil {
			errs = append(errs, fmt.Errorf("%s/%s: %w", release.Name, release.Environment.Name, err))
		}
	}

	return xerr.MultiErrOrderedFrom("validating releases", errs...)
}

type ValidateReleaseParams struct {
	Release      *v1alpha1.Release
	ValueMapping *config.ValueMapping
	Chart        *helm.ChartFS
	Helm         helm.PullRenderer
}

func ValidateRelease(ctx context.Context, params ValidateReleaseParams) error {
	if !params.Release.Environment.Spec.Promotion.FromPullRequests {
		version := "v" + params.Release.Spec.Version
		if semver.Prerelease(version)+semver.Build(version) != "" {
			return fmt.Errorf("invalid version: prerelease branches not allowed: %s", params.Release.Spec.Version)
		}
	}

	if yml.HasLockedTodos(params.Release.File.Tree) {
		return errors.New("contains locked TODO")
	}

	renderOpts := render.RenderReleaseParams{
		Release: params.Release,
		Chart:   params.Chart,
		CommonRenderParams: render.CommonRenderParams{
			ValueMapping: params.ValueMapping,
			IO: internal.IO{
				Out: io.Discard,
				Err: io.Discard,
				In:  io.NopCloser(strings.NewReader("")),
			},
			Helm: params.Helm,
		},
	}

	if err := render.RenderRelease(ctx, renderOpts); err != nil {
		return err
	}

	return nil
}
