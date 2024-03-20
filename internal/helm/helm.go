package helm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"path"

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
	ChartURL  string
	Version   string
	OutputDir string
}

func (cli CLI) Pull(ctx context.Context, opts PullOptions) error {
	chartURL, err := url.Parse(opts.ChartURL)
	if err != nil {
		return fmt.Errorf("invalid chart url: %w", err)
	}

	if opts.OutputDir == "" {
		opts.OutputDir = "."
	}

	if chartURL.Scheme == "" {
		chartURL.Scheme = "oci"
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

	if opts.Version != "" {
		args = append(args, "--version", opts.Version)
	}

	cmd := exec.CommandContext(ctx, "helm", args...)
	cmd.Stdout = cli.Out
	cmd.Stderr = cli.Err
	cmd.Stdin = cli.In

	return cmd.Run()
}

type RenderOpts struct {
	Dst         io.Writer
	ReleaseName string
	Values      map[string]any
	ChartPath   string
}

func (CLI) Render(ctx context.Context, opts RenderOpts) error {
	var input bytes.Buffer
	if err := yaml.NewEncoder(&input).Encode(opts.Values); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "helm", "template", opts.ReleaseName, opts.ChartPath, "--values", "-")
	cmd.Stdin = &input
	cmd.Stdout = opts.Dst
	cmd.Stderr = opts.Dst

	return cmd.Run()
}
