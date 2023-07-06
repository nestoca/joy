package yml

import (
	"bytes"
	"errors"
	"gopkg.in/yaml.v3"
	"reflect"
)

func EncodeYaml(obj any) ([]byte, error) {
	if reflect.ValueOf(obj).Kind() != reflect.Ptr {
		return nil, errors.New("obj must be a pointer")
	}

	var b bytes.Buffer

	yamlEncoder := yaml.NewEncoder(&b)
	yamlEncoder.SetIndent(2)
	err := yamlEncoder.Encode(obj)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
