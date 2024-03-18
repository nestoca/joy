package yml

import "os"

//go:generate moq -stub -out ./writer_mock.go . Writer
type Writer interface {
	WriteFile(file *File) error
}

type writerFunc func(file *File) error

func (fn writerFunc) WriteFile(file *File) error {
	return fn(file)
}

var DiskWriter = writerFunc(func(file *File) error {
	return os.WriteFile(file.Path, file.Yaml, 0o644)
})
