// Copyright 2017 HenryLee. All Rights Reserved.
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

// +build ignore

package xfer

import (
	"compress/gzip"
	"io"

	"github.com/henrylee2cn/teleport/utils"
)

// CodecWriter writer with gzip and encoder
type CodecWriter struct {
	io.Writer
	id            byte
	gzipWriterMap map[int]*gzip.Writer
	encMap        map[int]codec.Encoder
	encMaker      func(io.Writer) codec.Encoder
}

// Note: reseting the temporary buffer when return the *CodecWriter
func (s *socket) getCodecWriter(bodyCodec byte, w io.Writer) (*CodecWriter, error) {
	t, ok := s.codecWriterMap[codecId]
	if ok {
		t.Writer = w
		return t, nil
	}
	c, err := codec.GetByName(codecId)
	if err != nil {
		return nil, err
	}
	t = &CodecWriter{
		Writer:        w,
		id:            c.Id(),
		gzipWriterMap: s.gzipWriterMap,
		encMaker:      c.NewEncoder,
	}
	t.encMap = map[int]codec.Encoder{gzip.NoCompression: c.NewEncoder(t)}
	s.codecWriterMap[codecId] = t
	return t, nil
}

// Id returns codec id.
func (t *CodecWriter) Id() byte {
	return t.id
}

// Encode writes data with gzip and encoder.
func (t *CodecWriter) Encode(gzipLevel int, v interface{}) error {
	enc, ok := t.encMap[gzipLevel]
	if gzipLevel == gzip.NoCompression {
		return enc.Encode(v)
	}
	var gz *gzip.Writer
	var err error
	if ok {
		gz = t.gzipWriterMap[gzipLevel]
		gz.Reset(t)

	} else {
		gz, err = gzip.NewWriterLevel(t, gzipLevel)
		if err != nil {
			return err
		}
		t.gzipWriterMap[gzipLevel] = gz
		enc = t.encMaker(gz)
		t.encMap[gzipLevel] = enc
	}

	if err = enc.Encode(v); err != nil {
		return err
	}
	return gz.Flush()
}

type CodecReader struct {
	*utils.BufioReader
	name       string
	gzipReader *gzip.Reader
	dec        codec.Decoder
	gzDec      codec.Decoder
}

func (s *socket) makeCodecReader(codecId byte) (*CodecReader, error) {
	r, ok := s.codecReaderMap[codecId]
	if ok {
		return r, nil
	}
	c, err := codec.GetById(codecId)
	if err != nil {
		return nil, err
	}
	bufioReader := s.bufioReader
	gzipReader := s.gzipReader
	r = &CodecReader{
		BufioReader: bufioReader,
		gzipReader:  gzipReader,
		dec:         c.NewDecoder(bufioReader),
		gzDec:       c.NewDecoder(gzipReader),
		name:        c.Name(),
	}
	s.codecReaderMap[codecId] = r
	return r, nil
}

func (r *CodecReader) Name() string {
	return r.name
}

func (r *CodecReader) Decode(gzipLevel int, v interface{}) error {
	if gzipLevel == gzip.NoCompression {
		return r.dec.Decode(v)
	}
	var err error
	if err = r.gzipReader.Reset(r.BufioReader); err != nil {
		return err
	}
	return r.gzDec.Decode(v)
}
