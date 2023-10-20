package promote

import "github.com/nestoca/joy/internal/yml"

type FileSystemYamlWriter struct{}

func (w *FileSystemYamlWriter) Write(file *yml.File) error {
	return file.WriteYaml()
}
