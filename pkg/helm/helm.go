package helm

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/internal"
	"github.com/nestoca/joy/internal/observability"
)

type Puller interface {
	Pull(context.Context, PullOptions) error
}

//go:generate moq -stub -out ./pull_renderer_mock.go . PullRenderer
type PullRenderer interface {
	Puller
	Render(ctx context.Context, opts RenderOpts) (string, error)
}

type CLI struct {
	internal.IO
	Debug bool
}

type PullOptions struct {
	Chart     Chart
	OutputDir string
}

func (cli CLI) Pull(ctx context.Context, opts PullOptions) error {
	ctx, span := observability.StartTrace(ctx, "helm_pull")
	defer span.End()

	if opts.OutputDir == "" {
		opts.OutputDir = "."
	}

	chartURL, err := opts.Chart.ToURL()
	if err != nil {
		return fmt.Errorf("invalid chart url: %w", err)
	}

	var args []string

	switch chartURL.Scheme {
	case "http", "https":
		repo, chart := path.Split(chartURL.Path)
		chartURL.Path = repo
		args = []string{"pull", chart, "--repo", chartURL.String(), "--untar", "--untardir", opts.OutputDir}
	default:
		args = []string{"pull", chartURL.String(), "--untar", "--untardir", opts.OutputDir}
	}

	if version := opts.Chart.Version; version != "" {
		args = append(args, "--version", version)
	}

	cmd := exec.CommandContext(ctx, "helm", args...)
	cmd.Stdout = cli.Out
	cmd.Stderr = cli.Err
	cmd.Stdin = cli.In

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running %s: %w", strings.Join(cmd.Args, " "), err)
	}
	return nil
}

type RenderOpts struct {
	ReleaseName string
	Values      map[string]any
	ChartPath   string
	Namespace   string
}

func (cli CLI) Render(ctx context.Context, opts RenderOpts) (result string, err error) {
	ctx, span := observability.StartTrace(ctx, "helm_render")
	defer span.End()

	defer func() {
		if err != nil {
			span.RecordError(err)
		}
	}()

	var input bytes.Buffer
	if err := yaml.NewEncoder(&input).Encode(opts.Values); err != nil {
		return "", err
	}

	opts.ChartPath = strings.TrimPrefix(opts.ChartPath, "file://")

	args := []string{"template", opts.ReleaseName, opts.ChartPath, "--values", "-", "--skip-tests", "--namespace", cmp.Or(opts.Namespace, "default")}

	if cli.Debug {
		args = append(args, "--debug")
	}

	cmd := exec.CommandContext(ctx, "helm", args...)
	cmd.Stdin = &input

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, out)
	}

	return string(out), nil
}
