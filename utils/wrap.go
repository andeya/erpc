package utils

import (
	"io"
)

type WriterWrap struct {
	w     io.Writer
	count int
}

func (wp *WriterWrap) Write(p []byte) (int, error) {
	n, err := wp.w.Write(p)
	wp.count += n
	return n, err
}

func (wp *WriterWrap) Writed() int {
	return wp.count
}

func (wp *WriterWrap) Reset(w io.Writer) {
	wp.w = w
	wp.count = 0
}

type ReaderWrap struct {
	r     io.Reader
	count int
}

func (rp *ReaderWrap) Read(p []byte) (int, error) {
	n, err := rp.r.Read(p)
	rp.count += n
	return n, err
}

func (rp *ReaderWrap) Readed() int {
	return rp.count
}

func (rp *ReaderWrap) Reset(r io.Reader) {
	rp.r = r
	rp.count = 0
}
