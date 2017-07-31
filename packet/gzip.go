package packet

import (
	"compress/gzip"
	"io"

	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/utils"
)

type GzipEncoder struct {
	encMap   map[int]codec.Encoder
	gzMap    map[int]*gzip.Writer
	w        io.Writer
	encMaker codec.EncodeMaker
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
		encMap:   map[int]codec.Encoder{gzip.NoCompression: enc},
		gzMap:    make(map[int]*gzip.Writer),
		w:        w,
		encMaker: encMaker,
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
		gz = g.gzMap[gzipLevel]
		gz.Reset(g.w)

	} else {
		gz, err = gzip.NewWriterLevel(g.w, gzipLevel)
		if err != nil {
			return err
		}
		g.gzMap[gzipLevel] = gz
		enc = g.encMaker(gz)
		g.encMap[gzipLevel] = enc
	}

	if err = enc.Encode(v); err != nil {
		return err
	}
	return gz.Flush()
}

type GzipDecoder struct {
	dec      codec.Decoder
	gzDec    codec.Decoder
	gz       *gzip.Reader
	r        utils.Reader
	decMaker codec.DecodeMaker
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
	gz := c.gzipReader
	g = &GzipDecoder{
		dec:      dec,
		gzDec:    decMaker(gz),
		gz:       gz,
		r:        r,
		decMaker: decMaker,
	}
	c.gzipDecodeMap[codecName] = g
	return g, nil
}

func (g *GzipDecoder) Decode(gzipLevel int, v interface{}) error {
	if gzipLevel == gzip.NoCompression {
		return g.dec.Decode(v)
	}
	var err error
	if err = g.gz.Reset(g.r); err != nil {
		return err
	}
	return g.gzDec.Decode(v)
}
