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

	"github.com/henrylee2cn/erpc/v6/utils"
	"github.com/henrylee2cn/erpc/v6/xfer"
)

var ids = map[byte]bool{}

// Reg registers a gzip filter for transfer.
func Reg(id byte, name string, level int) {
	xfer.Reg(newGzip(id, name, level))
	ids[id] = true
}

// Is determines if the id is gzip.
func Is(id byte) bool {
	return ids[id]
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

// ID returns transfer filter id.
func (g *Gzip) ID() byte {
	return g.id
}

// ID returns transfer filter name.
func (g *Gzip) Name() string {
	return g.name
}

// OnPack performs filtering on packing.
func (g *Gzip) OnPack(src []byte) ([]byte, error) {
	bb := utils.AcquireByteBuffer()
	gw := g.wPool.Get().(*gzip.Writer)
	gw.Reset(bb)
	gw.Write(src)
	err := gw.Close()
	gw.Reset(nil)
	g.wPool.Put(gw)
	if err != nil {
		utils.ReleaseByteBuffer(bb)
		return nil, err
	}
	return bb.Bytes(), nil
}

// OnUnpack performs filtering on unpacking.
func (g *Gzip) OnUnpack(src []byte) (dest []byte, err error) {
	if len(src) == 0 {
		return src, nil
	}
	gr := g.rPool.Get().(*gzip.Reader)
	err = gr.Reset(bytes.NewReader(src))
	if err == nil {
		dest, err = ioutil.ReadAll(gr)
	}
	gr.Close()
	g.rPool.Put(gr)
	return dest, err
}
