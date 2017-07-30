package utils

import (
	"io"
)

// The actual read interface needed by NewReader.
// If the passed in io.Reader does not also have ReadByte,
// the NewReader will introduce its own buffering.
type Reader interface {
	io.Reader
	io.ByteReader
}

// LimitReader returns a Reader that reads from r
// but stops with EOF after n bytes.
// The underlying implementation is a *LimitedReader.
func LimitReader(r Reader, n int64) *LimitedReader { return &LimitedReader{r, n} }

// A LimitedBufioReader reads from bufio.Reader but limits the amount of
// data returned to just N bytes. Each call to Read
// updates N to reflect the new amount remaining.
// Read returns EOF when N <= 0 or when the underlying R returns EOF.
type LimitedReader struct {
	R Reader // underlying reader
	N int64  // max bytes remaining
}

func (l *LimitedReader) ResetLimit(n int64) {
	l.N = n
}

func (l *LimitedReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > l.N {
		p = p[0:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	return
}

func (l *LimitedReader) ReadByte() (byte, error) {
	if l.N <= 0 {
		return 0, io.EOF
	}
	l.N--
	return l.R.ReadByte()
}
