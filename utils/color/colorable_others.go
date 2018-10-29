// +build !windows

package color

import (
	"io"
	"os"
)

func NewColorable(file *os.File) io.Writer {
	if file == nil {
		panic("nil passed instead of *os.File to NewColorable()")
	}

	return file
}

func NewColorableStdout() io.Writer {
	return os.Stdout
}

func NewColorableStderr() io.Writer {
	return os.Stderr
}
