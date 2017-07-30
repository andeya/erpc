package utils

import (
	"compress/gzip"
	"io"
	"unsafe"
)

// A Writer is an io.WriteCloser.
// Writes to a Writer are compressed and written to w.
// Support unsafe way to reset the level.
type GzipWriter struct {
	*gzip.Writer
	levelPtr *int
}

// NewGzipWriter returns a new Writer.
// Writes to the returned writer are compressed and written to w.
//
// It is the caller's responsibility to call Close on the WriteCloser when done.
// Writes may be buffered and not flushed until Close.
//
// Callers that wish to set the fields in Writer.Header must do so before
// the first call to Write, Flush, or Close.
func NewGzipWriter(w io.Writer) *GzipWriter {
	z := gzip.NewWriter(w)
	addr := uintptr(unsafe.Pointer(z)) + unsafe.Sizeof(gzip.Header{}) + unsafe.Sizeof(io.Writer(nil))
	levelPtr := (*int)(unsafe.Pointer(addr))
	return &GzipWriter{
		Writer:   z,
		levelPtr: levelPtr,
	}
}

// ResetLevel unsafe way to reset the level.
func (w *GzipWriter) ResetLevel(level int) {
	*w.levelPtr = level
}
