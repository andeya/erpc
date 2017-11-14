// Copyright 2015-2017 HenryLee. All Rights Reserved.
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

type BufioReader struct {
	reader *bufio.Reader
	count  int64
	limit  int64 // max bytes remaining
}

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

func (b *BufioReader) ResetCount() {
	b.count = 0
}

func (b *BufioReader) ResetLimit(limit int64) {
	if limit < 0 {
		b.limit = math.MaxInt64
	} else {
		b.limit = limit
	}
}

func (b *BufioReader) Count() int64 {
	return b.count
}

func (b *BufioReader) Buffered() int {
	return b.reader.Buffered()
}

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

func (b *BufioReader) Reset(r io.Reader) {
	b.reader.Reset(r)
	b.count = 0
	b.limit = math.MaxInt64
}
