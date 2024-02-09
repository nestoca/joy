package helm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os/exec"

	"github.com/nestoca/joy/internal"
	"gopkg.in/yaml.v3"
)

//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE
type PullRenderer interface {
	Pull(context.Context, PullOptions) error
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

	if chartURL.Scheme == "" {
		chartURL.Scheme = "oci"
	}

	if opts.OutputDir == "" {
		opts.OutputDir = "."
	}

	args := []string{"pull", chartURL.String(), "--untar", "--untardir", opts.OutputDir}
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
	ChartPath   string
	Values      map[string]any
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
