package utils

import (
	"io"
)

// ReadWriteCounter reader and writer with counter
type ReadWriteCounter struct {
	*ReadCounter
	*WriteCounter
}

var _ io.ReadWriter = new(ReadWriteCounter)

// NewReadWriteCounter wrap the reader and writer with counter.
func NewReadWriteCounter(rw io.ReadWriter) *ReadWriteCounter {
	return &ReadWriteCounter{
		ReadCounter:  NewReadCounter(rw),
		WriteCounter: NewWriteCounter(rw),
	}
}

// Zero zero the counter.
func (rwp *ReadWriteCounter) Zero() {
	rwp.ReadCounter.Zero()
	rwp.WriteCounter.Zero()
}

// Reset resets itself.
func (rwp *ReadWriteCounter) Reset(rw io.ReadWriter) {
	rwp.ReadCounter.Reset(rw)
	rwp.WriteCounter.Reset(rw)
}

// ReadCounter reader with counter
type ReadCounter struct {
	r     io.Reader
	count int
}

var _ io.Reader = new(ReadCounter)

// NewReadCounter wrap the reader with counter.
func NewReadCounter(r io.Reader) *ReadCounter {
	return &ReadCounter{
		r: r,
	}
}

// Read reads bytes.
func (rp *ReadCounter) Read(p []byte) (int, error) {
	n, err := rp.r.Read(p)
	rp.count += n
	return n, err
}

// Readed returns readed bytes length.
func (rp *ReadCounter) Readed() int {
	return rp.count
}

// Zero zero the counter.
func (rp *ReadCounter) Zero() {
	rp.count = 0
}

// Reset resets itself.
func (rp *ReadCounter) Reset(r io.Reader) {
	rp.r = r
	rp.count = 0
}

// WriteCounter writer with counter
type WriteCounter struct {
	w     io.Writer
	count int
}

var _ io.Writer = new(WriteCounter)

// NewWriteCounter wrap the writer with counter.
func NewWriteCounter(w io.Writer) *WriteCounter {
	return &WriteCounter{
		w: w,
	}
}

// Write writes bytes.
func (wp *WriteCounter) Write(p []byte) (int, error) {
	n, err := wp.w.Write(p)
	wp.count += n
	return n, err
}

// Writed returns writed bytes length.
func (wp *WriteCounter) Writed() int {
	return wp.count
}

// Zero zero the counter.
func (wp *WriteCounter) Zero() {
	wp.count = 0
}

// Reset resets itself.
func (wp *WriteCounter) Reset(w io.Writer) {
	wp.w = w
	wp.count = 0
}
