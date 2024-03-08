package yml

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileUpdatesPreserveComments(t *testing.T) {
	file, err := NewFile("test.yml", []byte(`{
		# comment
		hello: world
	}`))
	require.NoError(t, err)

	require.NoError(t, SetOrAddNodeValue(file.Tree, "hello", "moon"))
	require.NoError(t, file.UpdateYamlFromTree())

	output, err := file.ToYaml()
	require.NoError(t, err)

	require.Equal(t, "{\n  # comment\n  hello: moon}\n", output)
}
