package yml_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/internal/yml"
)

const yamlString = `# Some random comment
apiVersion: joy.nesto.ca/v1alpha1
kind: Release
# another random comment
metadata:
    annotations: {}
    name: podinfo-deployment
spec:
    chart:
        name: podinfo
        # a nested comment
        repoUrl: https://stefanprodan.github.io/podinfo
        version: 6.3.6
    project: podinfo
    version: 1.0.0-avvvvvvd # This line will be modified in TestModifyingNodePreservesDocumentStructureAndOrdering
    versionKey: image.tag
`

func TestFindNodeInDocumentNodeWhenPathExists(t *testing.T) {
	yamlNode := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlString), yamlNode)
	assert.NoError(t, err)

	node, err := yml.FindNode(yamlNode, ".spec.chart.name")
	assert.NoError(t, err)
	assert.NotNil(t, node)

	assert.Equal(t, "podinfo", node.Value)
}

func TestFindNodeInMappingNodeWhenPathExists(t *testing.T) {
	yamlNode := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlString), yamlNode)
	assert.NoError(t, err)

	mappingNode := yamlNode.Content[0]
	assert.Equal(t, yaml.MappingNode, mappingNode.Kind)

	node, err := yml.FindNode(mappingNode, ".spec.chart.name")
	assert.NoError(t, err)
	assert.NotNil(t, node)

	assert.Equal(t, "podinfo", node.Value)
}

func TestFindNodeWhenPathDoesNotExist(t *testing.T) {
	yamlNode := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlString), yamlNode)
	assert.NoError(t, err)

	node, err := yml.FindNode(yamlNode, "spec.chart.name.foobar")
	assert.NotNil(t, err)
	assert.Nil(t, node)

	assert.EqualError(t, err, "node not found for path 'spec.chart.name.foobar': key 'name' does not exist")
}

func TestModifyingNodePreservesDocumentStructureAndOrdering(t *testing.T) {
	expected := `# Some random comment
apiVersion: joy.nesto.ca/v1alpha1
kind: Release
# another random comment
metadata:
    annotations: {}
    name: podinfo-deployment
spec:
    chart:
        name: podinfo
        # a nested comment
        repoUrl: https://stefanprodan.github.io/podinfo
        version: 6.3.6
    project: podinfo
    version: 1.0.0 # This line will be modified in TestModifyingNodePreservesDocumentStructureAndOrdering
    versionKey: image.tag
`

	yamlNode := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlString), yamlNode)
	assert.NoError(t, err)

	node, err := yml.FindNode(yamlNode, ".spec.version")
	assert.NoError(t, err)
	assert.NotNil(t, node)
	assert.Equal(t, "1.0.0-avvvvvvd", node.Value)

	node.Value = "1.0.0"

	rawBytes, err := yaml.Marshal(yamlNode)
	assert.Equal(t, expected, string(rawBytes))
}

func TestSetOrAddNodeValue_AddMissingNodesAndValue(t *testing.T) {
	yamlNode := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlString), yamlNode)
	assert.NoError(t, err)

	key := "metadata.annotations.abc.def"
	value := "test value"
	err = yml.SetOrAddNodeValue(yamlNode, key, value)
	assert.NoError(t, err)

	actualValue := yml.FindNodeValueOrDefault(yamlNode, key, "")
	assert.NoError(t, err)
	assert.Equal(t, value, actualValue)
}

func TestSetOrAddNodeValue_SetValueOfExistingNode(t *testing.T) {
	yamlNode := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlString), yamlNode)
	assert.NoError(t, err)

	key := "metadata.name"
	value := "test value"
	err = yml.SetOrAddNodeValue(yamlNode, key, value)
	assert.NoError(t, err)

	actualValue := yml.FindNodeValueOrDefault(yamlNode, key, "")
	assert.NoError(t, err)
	assert.Equal(t, value, actualValue)
}
