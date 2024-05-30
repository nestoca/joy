package yml

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

func UnmarshalStrict(data []byte, ptr any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	return decoder.Decode(ptr)
}
