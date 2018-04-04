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
	"io"
)

// BufioWriter implements buffering for an io.Writer object with count.
// If an error occurs writing to a BufioWriter, no more data will be
// accepted and all subsequent writes, and Flush, will return the error.
// After all data has been written, the client should call the
// Flush method to guarantee all data has been forwarded to
// the underlying io.Writer.
type BufioWriter struct {
	writer *bufio.Writer
	count  int64
}

// NewBufioWriter returns a new BufioWriter whose buffer has the default size.
func NewBufioWriter(w io.Writer) *BufioWriter {
	return &BufioWriter{
		writer: bufio.NewWriter(w),
	}
}

// NewBufioWriterSize returns a new BufioWriter whose buffer has at least the specified
// size. If the argument io.Writer is already a BufioWriter with large enough
// size, it returns the underlying BufioWriter.
func NewBufioWriterSize(w io.Writer, size int) *BufioWriter {
	return &BufioWriter{
		writer: bufio.NewWriterSize(w, size),
	}
}

// ResetCount resets the count.
func (b *BufioWriter) ResetCount() {
	b.count = 0
}

// Count returns the count.
func (b *BufioWriter) Count() int64 {
	return b.count
}

// Available returns how many bytes are unused in the buffer.
func (b *BufioWriter) Available() int {
	return b.writer.Available()
}

// Buffered returns the number of bytes that have been written into the current buffer.
func (b *BufioWriter) Buffered() int {
	return b.writer.Buffered()
}

// Flush writes any buffered data to the underlying io.Writer.
func (b *BufioWriter) Flush() error {
	return b.writer.Flush()
}

// ReadFrom implements io.ReaderFrom.
func (b *BufioWriter) ReadFrom(r io.Reader) (int64, error) {
	n, err := b.writer.ReadFrom(r)
	b.count += n
	return n, err
}

// Reset discards any unflushed buffered data, clears any error,
// resets b to write its output to w, and resets count.
func (b *BufioWriter) Reset(w io.Writer) {
	b.writer.Reset(w)
	b.count = 0
}

// Write writes the contents of p into the buffer.
// It returns the number of bytes written.
// If nn < len(p), it also returns an error explaining
// why the write is short.
func (b *BufioWriter) Write(p []byte) (int, error) {
	n, err := b.writer.Write(p)
	b.count += int64(n)
	return n, err
}

// WriteByte writes a single byte.
func (b *BufioWriter) WriteByte(c byte) error {
	err := b.writer.WriteByte(c)
	if err == nil {
		b.count++
	}
	return err
}

// WriteRune writes a single Unicode code point, returning
// the number of bytes written and any error.
func (b *BufioWriter) WriteRune(r rune) (int, error) {
	size, err := b.writer.WriteRune(r)
	b.count += int64(size)
	return size, err
}

// WriteString writes a string.
// It returns the number of bytes written.
// If the count is less than len(s), it also returns an error explaining
// why the write is short.
func (b *BufioWriter) WriteString(s string) (int, error) {
	n, err := b.writer.WriteString(s)
	b.count += int64(n)
	return n, err
}
