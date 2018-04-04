// Copyright 2015-2018 HenryLee. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"bufio"
	"compress/flate"
	"io"
	"math"
)

var _ flate.Reader = new(BufioReader)

// BufioReader an io.Reader object buffer with count and limit.
type BufioReader struct {
	reader *bufio.Reader
	count  int64
	limit  int64 // max bytes remaining
}

// NewBufioReader returns a new BufioReader whose buffer has the default size.
func NewBufioReader(r io.Reader, limit ...int64) *BufioReader {
	br := &BufioReader{
		reader: bufio.NewReader(r),
	}
	if len(limit) > 0 && limit[0] >= 0 {
		br.limit = limit[0]
	} else {
		br.limit = math.MaxInt64
	}
	return br
}

// NewBufioReaderSize returns a new BufioReader whose buffer has at least the specified
// size. If the argument io.Reader is already a BufioReader with large enough
// size, it returns the underlying BufioReader.
func NewBufioReaderSize(r io.Reader, size int, limit ...int64) *BufioReader {
	br := &BufioReader{
		reader: bufio.NewReaderSize(r, size),
	}
	if len(limit) > 0 {
		br.limit = limit[0]
	} else {
		br.limit = math.MaxInt64
	}
	return br
}

// ResetCount resets the count.
func (b *BufioReader) ResetCount() {
	b.count = 0
}

// ResetLimit resets the limit.
func (b *BufioReader) ResetLimit(limit int64) {
	if limit < 0 {
		b.limit = math.MaxInt64
	} else {
		b.limit = limit
	}
}

// Count returns the count.
func (b *BufioReader) Count() int64 {
	return b.count
}

// Buffered returns the number of bytes that can be read from the current buffer.
func (b *BufioReader) Buffered() int {
	return b.reader.Buffered()
}

// Discard skips the next n bytes, returning the number of bytes discarded.
//
// If Discard skips fewer than n bytes, it also returns an error.
// If 0 <= n <= b.Buffered(), Discard is guaranteed to succeed without
// reading from the underlying io.Reader.
func (b *BufioReader) Discard(n int) (discarded int, err error) {
	if b.limit <= 0 {
		return 0, io.EOF
	}
	if b.limit < int64(n) {
		n = int(b.limit)
	}
	discarded, err = b.reader.Discard(n)
	count := int64(discarded)
	b.count += count
	b.limit -= count
	return
}

// Read reads data into p.
// It returns the number of bytes read into p.
// The bytes are taken from at most one Read on the underlying Reader,
// hence n may be less than len(p).
// At EOF, the count will be zero and err will be io.EOF.
func (b *BufioReader) Read(p []byte) (int, error) {
	if b.limit <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > b.limit {
		p = p[0:b.limit]
	}
	n, err := b.reader.Read(p)
	count := int64(n)
	b.count += count
	b.limit -= count
	return n, err
}

// ReadByte reads and returns a single byte.
// If no byte is available, returns an error.
func (b *BufioReader) ReadByte() (byte, error) {
	if b.limit <= 0 {
		return 0, io.EOF
	}
	a, err := b.reader.ReadByte()
	if err == nil {
		b.count++
		b.limit--
	}
	return a, err
}

// Reset discards any buffered data, resets all state, switches
// the buffered reader to read from r, resets count, and resets limit.
func (b *BufioReader) Reset(r io.Reader) {
	b.reader.Reset(r)
	b.count = 0
	b.limit = math.MaxInt64
}
