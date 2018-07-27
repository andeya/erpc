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

package gzip

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/henrylee2cn/teleport/utils"
	"github.com/henrylee2cn/teleport/xfer"
)

// Reg registers a gzip filter for transfer.
func Reg(id byte, name string, level int) {
	xfer.Reg(newGzip(id, name, level))
}

// newGzip creates a new gizp filter.
func newGzip(id byte, name string, level int) *Gzip {
	if level < gzip.HuffmanOnly || level > gzip.BestCompression {
		panic(fmt.Sprintf("gzip: invalid compression level: %d", level))
	}
	g := new(Gzip)
	g.level = level
	g.id = id
	g.name = name
	g.wPool = sync.Pool{
		New: func() interface{} {
			gw, _ := gzip.NewWriterLevel(nil, g.level)
			return gw
		},
	}
	g.rPool = sync.Pool{
		New: func() interface{} {
			return new(gzip.Reader)
		},
	}
	return g
}

// Gzip compression filter
type Gzip struct {
	id    byte
	name  string
	level int
	wPool sync.Pool
	rPool sync.Pool
}

// Id returns transfer filter id.
func (g *Gzip) Id() byte {
	return g.id
}

// Id returns transfer filter name.
func (g *Gzip) Name() string {
	return g.name
}

// OnPack performs filtering on packing.
func (g *Gzip) OnPack(src []byte) ([]byte, error) {
	gw := g.wPool.Get().(*gzip.Writer)
	defer g.wPool.Put(gw)
	bb := utils.AcquireByteBuffer()
	gw.Reset(bb)
	_, err := gw.Write(src)
	if err != nil {
		utils.ReleaseByteBuffer(bb)
		return nil, err
	}
	err = gw.Flush()
	if err != nil {
		utils.ReleaseByteBuffer(bb)
		return nil, err
	}
	return bb.Bytes(), nil
}

// OnUnpack performs filtering on unpacking.
func (g *Gzip) OnUnpack(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	gr := g.rPool.Get().(*gzip.Reader)
	defer g.rPool.Put(gr)
	err := gr.Reset(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	dest, _ := ioutil.ReadAll(gr)
	return dest, nil
}
