package catalog

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/internal/yml"
)

func TestValidateTagsForFiles(t *testing.T) {
	type File struct {
		Path    string
		Content string
	}

	cases := []struct {
		Name        string
		Files       []File
		ExpectedErr string
	}{
		{
			Name: "no unknown tags",
			Files: []File{
				{
					Path:    "./explicit.yaml",
					Content: "{hello: !!str world, answer: !!int 42, happy: !!bool true, pi: !!float 3.14, arr: !!seq [], dict: !!map {}}",
				},
				{
					Path:    "./implicit.yaml",
					Content: "{hello: world, answer: 42, happy: true, pi: 3.14, arr: [], dict: {}}",
				},
				{
					Path:    "./custom.yaml",
					Content: "{pad: !lock ''}",
				},
			},
		},
		{
			Name: "unknown tags",
			Files: []File{
				{
					Path:    "./a.yaml",
					Content: "{secret: !shhh {}, dont: !look {}}",
				},
				{
					Path:    "./b.yaml",
					Content: "{hello: !!woah {}}",
				},
			},
			ExpectedErr: "./a.yaml: unknown tag(s): !look, !shhh\n./b.yaml: unknown tag(s): !!woah",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			files := make([]*yml.File, len(tc.Files))
			for i, file := range tc.Files {
				var node yaml.Node
				require.NoError(t, yaml.Unmarshal([]byte(file.Content), &node))
				files[i] = &yml.File{Path: file.Path, Tree: &node}
			}

			err := validateTagsForFiles(files)
			if tc.ExpectedErr != "" {
				require.EqualError(t, err, tc.ExpectedErr)
				return
			}

			require.NoError(t, err)
		})
	}
}
