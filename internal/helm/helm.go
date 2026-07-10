package helm

import (
	"net/url"

	"github.com/davidmdm/x/xfs"
)

type Chart struct {
	RepoURL  string         `json:"repoUrl" yaml:"repoUrl"`
	Name     string         `json:"name" yaml:"name"`
	Version  string         `json:"version" yaml:"version"`
	Mappings map[string]any `json:"mappings,omitempty" yaml:"mappings"`
}

func (chart Chart) ToURL() (*url.URL, error) {
	uri, err := url.Parse(chart.RepoURL)
	if err != nil {
		return nil, err
	}

	if uri.Scheme == "" {
		uri.Scheme = "oci"
		uri, err = url.Parse(uri.String())
		if err != nil {
			return nil, err
		}
	}

	return uri.JoinPath(chart.Name), nil
}

type ChartFS struct {
	Chart
	xfs.FS
}
