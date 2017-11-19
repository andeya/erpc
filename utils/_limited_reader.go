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
	"compress/flate"
	"io"
	"math"
)

var _ flate.Reader = new(LimitedReader)

type LimitedReader struct {
	r     io.Reader
	count int64
	limit int64 // max bytes remaining
}

func NewLimitedReader(r io.Reader, limit ...int64) *LimitedReader {
	br := &LimitedReader{
		r: r,
	}
	if len(limit) > 0 && limit[0] >= 0 {
		br.limit = limit[0]
	} else {
		br.limit = math.MaxInt64
	}
	return br
}

func (b *LimitedReader) Reset(r io.Reader) {
	b.r = r
	b.count = 0
	b.limit = math.MaxInt64
}

func (b *LimitedReader) ResetCount() {
	b.count = 0
}

func (b *LimitedReader) ResetLimit(limit int64) {
	if limit < 0 {
		b.limit = math.MaxInt64
	} else {
		b.limit = limit
	}
}

func (b *LimitedReader) Count() int64 {
	return b.count
}

func (b *LimitedReader) Read(p []byte) (int, error) {
	if b.limit <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > b.limit {
		p = p[0:b.limit]
	}
	n, err := b.r.Read(p)
	count := int64(n)
	b.count += count
	b.limit -= count
	return n, err
}

func (b *LimitedReader) ReadByte() (byte, error) {
	if b.limit <= 0 {
		return 0, io.EOF
	}
	var (
		a   byte
		err error
	)
	if rr, ok := b.r.(flate.Reader); ok {
		a, err = rr.ReadByte()
	} else {
		var aa = make([]byte, 1)
		_, err = b.r.Read(aa)
		a = aa[0]
	}
	if err == nil {
		b.count++
		b.limit--
	}
	return a, err
}
