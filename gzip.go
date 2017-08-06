// Copyright 2016 HenryLee. All Rights Reserved.
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

package teleport

import (
	"compress/gzip"
	"io"

	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/utils"
)

type GzipEncoder struct {
	gzipWriterMap map[int]*gzip.Writer
	w             io.Writer
	encMap        map[int]codec.Encoder
	encMaker      codec.EncodeMaker
}

func (c *conn) getGzipEncoder(codecName string) (*GzipEncoder, error) {
	g, ok := c.gzipEncodeMap[codecName]
	if ok {
		return g, nil
	}
	encMaker, err := codec.GetEncodeMaker(codecName)
	if err != nil {
		return nil, err
	}
	w := c.cacheWriter
	enc := encMaker(w)
	g = &GzipEncoder{
		gzipWriterMap: c.gzipWriterMap,
		w:             w,
		encMap:        map[int]codec.Encoder{gzip.NoCompression: enc},
		encMaker:      encMaker,
	}
	c.gzipEncodeMap[codecName] = g
	return g, nil
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
	gzipReader *gzip.Reader
	r          utils.Reader
	dec        codec.Decoder
	gzDec      codec.Decoder
	decMaker   codec.DecodeMaker
}

func (c *conn) getGzipDecoder(codecName string) (*GzipDecoder, error) {
	g, ok := c.gzipDecodeMap[codecName]
	if ok {
		return g, nil
	}
	decMaker, err := codec.GetDecodeMaker(codecName)
	if err != nil {
		return nil, err
	}
	r := c.limitReader
	dec := decMaker(r)
	gzipReader := c.gzipReader
	g = &GzipDecoder{
		dec:        dec,
		gzDec:      decMaker(gzipReader),
		gzipReader: gzipReader,
		r:          r,
		decMaker:   decMaker,
	}
	c.gzipDecodeMap[codecName] = g
	return g, nil
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
