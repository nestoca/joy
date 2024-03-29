package helm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/internal"
)

type Puller interface {
	Pull(context.Context, PullOptions) error
}

//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE
type PullRenderer interface {
	Puller
	Render(ctx context.Context, opts RenderOpts) error
}

type CLI struct {
	internal.IO
}

type PullOptions struct {
	Chart     Chart
	OutputDir string
}

func (cli CLI) Pull(ctx context.Context, opts PullOptions) error {
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
	Dst         io.Writer
	ReleaseName string
	Values      map[string]any
	ChartPath   string
}

func (cli CLI) Render(ctx context.Context, opts RenderOpts) error {
	var input bytes.Buffer
	if err := yaml.NewEncoder(&input).Encode(opts.Values); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "helm", "template", opts.ReleaseName, opts.ChartPath, "--values", "-")
	cmd.Stdin = &input
	cmd.Stdout = opts.Dst
	cmd.Stderr = cli.IO.Err

	return cmd.Run()
}
