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
	Render(context.Context, io.Writer, string, map[string]any) error
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

func (CLI) Render(ctx context.Context, w io.Writer, chartPath string, values map[string]any) error {
	var input bytes.Buffer
	if err := yaml.NewEncoder(&input).Encode(values); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "helm", "template", "joy-release-render", chartPath, "--values", "-")
	cmd.Stdin = &input
	cmd.Stdout = w
	cmd.Stderr = w

	return cmd.Run()
}
