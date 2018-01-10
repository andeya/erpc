package utils

import (
	"io"
)

// WriterWrap writer wrapper
type WriterWrap struct {
	w     io.Writer
	count int
}

// Write writes bytes.
func (wp *WriterWrap) Write(p []byte) (int, error) {
	n, err := wp.w.Write(p)
	wp.count += n
	return n, err
}

// Writed returns writed bytes length.
func (wp *WriterWrap) Writed() int {
	return wp.count
}

// Reset resets itself.
func (wp *WriterWrap) Reset(w io.Writer) {
	wp.w = w
	wp.count = 0
}

// ReaderWrap reader wrapper
type ReaderWrap struct {
	r     io.Reader
	count int
}

// Read reads bytes.
func (rp *ReaderWrap) Read(p []byte) (int, error) {
	n, err := rp.r.Read(p)
	rp.count += n
	return n, err
}

// Readed returns readed bytes length.
func (rp *ReaderWrap) Readed() int {
	return rp.count
}

// Reset resets itself.
func (rp *ReaderWrap) Reset(r io.Reader) {
	rp.r = r
	rp.count = 0
}
