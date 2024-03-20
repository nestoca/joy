package config

import (
	"testing"

	"github.com/go-test/deep"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestValueMappingUnmarshalling(t *testing.T) {
	cases := []struct {
		Name     string
		Input    string
		Expected ValueMapping
	}{
		{
			Name:  "mapping only",
			Input: "{key: value}",
			Expected: ValueMapping{
				ReleaseIgnoreList: nil,
				Mappings:          map[string]any{"key": "value"},
			},
		},
		{
			Name:  "mapping and ignore list",
			Input: "{releaseIgnoreList: [test], mappings: {key: value}}",
			Expected: ValueMapping{
				ReleaseIgnoreList: []string{"test"},
				Mappings:          map[string]any{"key": "value"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			var output ValueMapping
			require.NoError(t, yaml.Unmarshal([]byte(tc.Input), &output))
			require.Equal(t, tc.Expected, output, "%#v", output)
		})
	}
}

func TestDeepCopyConfigNonZeroValues(t *testing.T) {
	getNonZeroValues1 := func() *Config {
		return &Config{
			MinVersion:   "1.0.0",
			DefaultChart: "chart1",
			ValueMapping: &ValueMapping{
				ReleaseIgnoreList: []string{"ignore1"},
				Mappings:          map[string]any{"key1": "value1"},
			},
			ReferenceEnvironment: "ref1",
			GitHubOrganization:   "org1",
			Templates: Templates{
				Project: ProjectTemplates{
					GitTag: "tag1",
				},
				Release: ReleaseTemplates{
					Promote: ReleasePromoteTemplates{
						Commit:      "commit1",
						PullRequest: "pr1",
					},
					Links: map[string]string{
						"link1": "url1",
					},
				},
			},
		}
	}
	getNonZeroValues2 := func() *Config {
		return &Config{
			MinVersion:   "2.0.0",
			DefaultChart: "chart2",
			ValueMapping: &ValueMapping{
				ReleaseIgnoreList: []string{"ignore2"},
				Mappings:          map[string]any{"key2": "value2"},
			},
			ReferenceEnvironment: "ref2",
			GitHubOrganization:   "org2",
			Templates: Templates{
				Project: ProjectTemplates{
					GitTag: "tag2",
				},
				Release: ReleaseTemplates{
					Promote: ReleasePromoteTemplates{
						Commit:      "commit2",
						PullRequest: "pr2",
					},
					Links: map[string]string{
						"link2": "url2",
					},
				},
			},
		}
	}
	getMergedValues := func() *Config {
		return &Config{
			MinVersion:   "2.0.0",
			DefaultChart: "chart2",
			ValueMapping: &ValueMapping{
				ReleaseIgnoreList: []string{"ignore2"},
				Mappings:          map[string]any{"key2": "value2"},
			},
			ReferenceEnvironment: "ref2",
			GitHubOrganization:   "org2",
			Templates: Templates{
				Project: ProjectTemplates{
					GitTag: "tag2",
				},
				Release: ReleaseTemplates{
					Promote: ReleasePromoteTemplates{
						Commit:      "commit2",
						PullRequest: "pr2",
					},
					Links: map[string]string{
						"link1": "url1",
						"link2": "url2",
					},
				},
			},
		}
	}
	getZeroValues := func() *Config {
		return &Config{}
	}
	cases := []struct {
		Name        string
		GetSource   func() *Config
		GetTarget   func() *Config
		GetExpected func() *Config
	}{
		{
			Name:        "Both source and target non-zero",
			GetSource:   getNonZeroValues2,
			GetTarget:   getNonZeroValues1,
			GetExpected: getMergedValues,
		},
		{
			Name:        "Source zero, target non-zero",
			GetSource:   getZeroValues,
			GetTarget:   getNonZeroValues1,
			GetExpected: getNonZeroValues1,
		},
		{
			Name:        "Source non-zero, target zero",
			GetSource:   getNonZeroValues1,
			GetTarget:   getZeroValues,
			GetExpected: getNonZeroValues1,
		},
		{
			Name:        "Both source and target zero",
			GetSource:   getZeroValues,
			GetTarget:   getZeroValues,
			GetExpected: getZeroValues,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			source := tc.GetSource()
			target := tc.GetTarget()
			deepCopyConfigNonZeroValues(source, target)
			diff := deep.Equal(tc.GetExpected(), target)
			if diff != nil {
				require.Fail(t, "DeepEqual failed", "diff: %v", diff)
			}
		})
	}
}
