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

package socket

import (
	"compress/gzip"
	"io"

	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/utils"
)

type GzipEncoder struct {
	id            byte
	gzipWriterMap map[int]*gzip.Writer
	w             io.Writer
	encMap        map[int]codec.Encoder
	encMaker      func(io.Writer) codec.Encoder
}

func (s *socket) getGzipEncoder(codecName string) (*GzipEncoder, error) {
	g, ok := s.gzipEncodeMap[codecName]
	if ok {
		return g, nil
	}
	c, err := codec.GetByName(codecName)
	if err != nil {
		return nil, err
	}
	w := s.cacheWriter
	enc := c.NewEncoder(w)
	g = &GzipEncoder{
		id:            c.Id(),
		gzipWriterMap: s.gzipWriterMap,
		w:             w,
		encMap:        map[int]codec.Encoder{gzip.NoCompression: enc},
		encMaker:      c.NewEncoder,
	}
	s.gzipEncodeMap[codecName] = g
	return g, nil
}

func (g *GzipEncoder) Id() byte {
	return g.id
}

func (g *GzipEncoder) Encode(gzipLevel int, v interface{}) error {
	enc, ok := g.encMap[gzipLevel]
	if gzipLevel == gzip.NoCompression {
		return enc.Encode(v)
	}
	var gz *gzip.Writer
	var err error
	if ok {
		gz = g.gzipWriterMap[gzipLevel]
		gz.Reset(g.w)

	} else {
		gz, err = gzip.NewWriterLevel(g.w, gzipLevel)
		if err != nil {
			return err
		}
		g.gzipWriterMap[gzipLevel] = gz
		enc = g.encMaker(gz)
		g.encMap[gzipLevel] = enc
	}

	if err = enc.Encode(v); err != nil {
		return err
	}
	return gz.Flush()
}

type GzipDecoder struct {
	name       string
	gzipReader *gzip.Reader
	r          utils.Reader
	dec        codec.Decoder
	gzDec      codec.Decoder
	decMaker   func(io.Reader) codec.Decoder
}

func (s *socket) getGzipDecoder(codecId byte) (*GzipDecoder, error) {
	g, ok := s.gzipDecodeMap[codecId]
	if ok {
		return g, nil
	}
	c, err := codec.GetById(codecId)
	if err != nil {
		return nil, err
	}
	r := s.limitReader
	gzipReader := s.gzipReader
	g = &GzipDecoder{
		r:          r,
		gzipReader: gzipReader,
		decMaker:   c.NewDecoder,
		dec:        c.NewDecoder(r),
		gzDec:      c.NewDecoder(gzipReader),
		name:       c.Name(),
	}
	s.gzipDecodeMap[codecId] = g
	return g, nil
}

func (g *GzipDecoder) Name() string {
	return g.name
}

func (g *GzipDecoder) Decode(gzipLevel int, v interface{}) error {
	if gzipLevel == gzip.NoCompression {
		return g.dec.Decode(v)
	}
	var err error
	if err = g.gzipReader.Reset(g.r); err != nil {
		return err
	}
	return g.gzDec.Decode(v)
}
