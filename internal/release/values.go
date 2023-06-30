package release

type Values struct {
	// FilePath is the path to the values file.
	FilePath string `yaml:"-"`

	// Yaml is the raw yaml of the values file.
	Yaml string `yaml:"-"`
}
