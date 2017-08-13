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
	"io"
)

type BufioWriter struct {
	writer *bufio.Writer
	count  int64
}

func NewBufioWriter(w io.Writer) *BufioWriter {
	return &BufioWriter{
		writer: bufio.NewWriter(w),
	}
}

func NewBufioWriterSize(w io.Writer, size int) *BufioWriter {
	return &BufioWriter{
		writer: bufio.NewWriterSize(w, size),
	}
}

func (b *BufioWriter) ResetCount() {
	b.count = 0
}

func (b *BufioWriter) Count() int64 {
	return b.count
}

func (b *BufioWriter) Available() int {
	return b.writer.Available()
}

func (b *BufioWriter) Buffered() int {
	return b.writer.Buffered()
}

func (b *BufioWriter) Flush() error {
	return b.writer.Flush()
}

func (b *BufioWriter) ReadFrom(r io.Reader) (int64, error) {
	n, err := b.writer.ReadFrom(r)
	b.count += n
	return n, err
}

func (b *BufioWriter) Reset(w io.Writer) {
	b.writer.Reset(w)
	b.count = 0
}

func (b *BufioWriter) Write(p []byte) (int, error) {
	n, err := b.writer.Write(p)
	b.count += int64(n)
	return n, err
}

func (b *BufioWriter) WriteByte(c byte) error {
	err := b.writer.WriteByte(c)
	if err == nil {
		b.count++
	}
	return err
}

func (b *BufioWriter) WriteRune(r rune) (int, error) {
	size, err := b.writer.WriteRune(r)
	b.count += int64(size)
	return size, err
}

func (b *BufioWriter) WriteString(s string) (int, error) {
	n, err := b.writer.WriteString(s)
	b.count += int64(n)
	return n, err
}

type BufioReader struct {
	reader *bufio.Reader
	count  int64
}

func NewBufioReader(r io.Reader) *BufioReader {
	return &BufioReader{
		reader: bufio.NewReader(r),
	}
}

func NewBufioReaderSize(r io.Reader, size int) *BufioReader {
	return &BufioReader{
		reader: bufio.NewReaderSize(r, size),
	}
}

func (b *BufioReader) ResetCount() {
	b.count = 0
}

func (b *BufioReader) Count() int64 {
	return b.count
}

func (b *BufioReader) Buffered() int {
	return b.reader.Buffered()
}

func (b *BufioReader) Discard(n int) (discarded int, err error) {
	discarded, err = b.reader.Discard(n)
	b.count += int64(discarded)
	return
}

func (b *BufioReader) Peek(n int) ([]byte, error) {
	a, err := b.reader.Peek(n)
	b.count += int64(len(a))
	return a, err
}

func (b *BufioReader) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	b.count += int64(n)
	return n, err
}

func (b *BufioReader) ReadByte() (byte, error) {
	a, err := b.reader.ReadByte()
	if err == nil {
		b.count++
	}
	return a, err
}

func (b *BufioReader) Reset(r io.Reader) {
	b.reader.Reset(r)
	b.count = 0
}

func (b *BufioReader) WriteTo(w io.Writer) (int64, error) {
	n, err := b.reader.WriteTo(w)
	b.count += n
	return n, err
}
