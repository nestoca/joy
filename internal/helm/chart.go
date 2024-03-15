package helm

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/davidmdm/x/xerr"
	"gopkg.in/yaml.v3"
)

type Chart struct {
	URL     string `yaml:"url"`
	Version string `yaml:"version"`
}

func (chart *Chart) UnmarshalYAML(node *yaml.Node) (err error) {
	defer func() {
		if err != nil {
			return
		}

		err = xerr.MultiErrOrderedFrom(
			"invalid chart",
			func() error {
				if chart.URL == "" {
					return fmt.Errorf("url required")
				}
				return nil
			}(),
			func() error {
				if chart.Version == "" {
					return fmt.Errorf("version required")
				}
				return nil
			}(),
		)
	}()

	var rawURL string
	if err = node.Decode(&rawURL); err == nil {
		*chart, err = parseChartURL(rawURL)
		return
	}

	// We need to create an internal type so that Unmarshalling isn't recursive
	type ChartStructure Chart
	var values ChartStructure

	if err := node.Decode(&values); err != nil {
		return err
	}

	*chart = Chart(values)

	// parse the chart.URL so that it can benefit from default scheme and have its version stripped
	// Only the version property will be respected
	internalChart, err := parseChartURL(chart.URL)
	if err != nil {
		return err
	}
	chart.URL = internalChart.URL

	return nil
}

func parseChartURL(raw string) (Chart, error) {
	uri, err := url.Parse(raw)
	if err != nil {
		return Chart{}, err
	}

	chartPath, version, _ := strings.Cut(uri.Path, ":")
	if chartPath != "" && uri.Scheme == "" {
		uri.Scheme = "oci"
	}

	uri.Path = chartPath

	return Chart{
		URL:     uri.String(),
		Version: version,
	}, nil
}
