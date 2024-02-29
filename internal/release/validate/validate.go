package validate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/encoding/gocode/gocodec"

	"github.com/davidmdm/x/xerr"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/helm"
	"github.com/nestoca/joy/internal/release/render"
)

type ValidateParams struct {
	Releases     []*v1alpha1.Release
	DefaultChart string
	CacheRoot    string
	Helm         helm.PullRenderer
	ValueMapping *config.ValueMapping
}

func Validate(ctx context.Context, params ValidateParams) error {
	cache := helm.ChartCache{
		DefaultChart: params.DefaultChart,
		Root:         params.CacheRoot,
		Puller:       params.Helm,
	}

	var errs []error
	for _, release := range params.Releases {
		chart, err := cache.GetReleaseChart(ctx, release)
		if err != nil {
			return fmt.Errorf("getting release chart: %w", err)
		}

		validateParams := ValidateReleaseParams{
			Chart:        chart,
			Release:      release,
			DefaultChart: params.DefaultChart,
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
	DefaultChart string
	ValueMapping *config.ValueMapping
	Chart        *helm.Chart
	Helm         helm.PullRenderer
}

func ValidateRelease(ctx context.Context, params ValidateReleaseParams) error {
	if err := validateSchema(params.Release, params.Chart); err != nil {
		return err
	}

	// TODO AYA
	// Add a check that releases don't have pre-releae if the PromotionFromPullRequest is false

	renderOpts := render.RenderReleaseParams{
		Release: params.Release,
		Chart:   params.Chart,
		CommonRenderParams: render.CommonRenderParams{
			DefaultChart: params.DefaultChart,
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

func validateSchema(release *v1alpha1.Release, chart *helm.Chart) error {
	if release.Spec.Values == nil {
		return nil
	}

	schemaData, err := chart.ReadFile("values.cue")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("reading schema file: %w", err)
	}

	runtime := cuecontext.New()

	schema := runtime.
		CompileBytes(schemaData).
		LookupPath(cue.MakePath(cue.Def("#values")))

	codec := gocodec.New(runtime, &gocodec.Config{})

	if errs := cueerrors.Errors(codec.Validate(schema, release.Spec.Values)); len(errs) == 1 {
		return errs[0]
	} else if len(errs) > 1 {
		return xerr.MultiErrOrderedFrom(errs[0].Error(), AsErrorList(errs[1:])...)
	}

	return nil
}

func AsErrorList[T error](list []T) []error {
	result := make([]error, len(list))
	for i, err := range list {
		result[i] = err
	}
	return result
}
