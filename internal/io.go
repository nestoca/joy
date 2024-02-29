package internal

import (
	"io"
)

type IO struct {
	Out io.Writer
	Err io.Writer
	In  io.Reader
}
