// Socket package provides a concise, powerful and high-performance TCP
//
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
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"sync"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/teleport/utils"
)

type (
	// Proto pack/unpack protocol scheme of socket packet.
	Proto interface {
		// Version returns the protocol's id and name.
		Version() (byte, string)
		// Pack pack socket data packet.
		// Note: Make sure to write only once or there will be package contamination!
		Pack(*Packet) error
		// Unpack unpack socket data packet.
		// Note: Concurrent unsafe!
		Unpack(*Packet) error
	}
	ProtoFunc func(io.ReadWriter) Proto
)

// DefaultProtoFunc gets the default builder of socket communication protocol
func DefaultProtoFunc() ProtoFunc {
	return defaultProtoFunc
}

// SetDefaultProtoFunc sets the default builder of socket communication protocol
func SetDefaultProtoFunc(protoFunc ProtoFunc) {
	defaultProtoFunc = protoFunc
}

type (
	// FastProto fast socket communication protocol.
	FastProto struct {
		id   byte
		name string
		r    io.Reader
		w    io.Writer
		rMu  sync.Mutex
	}
)

// default builder of socket communication protocol
var (
	defaultProtoFunc = func(rw io.ReadWriter) Proto {
		return &FastProto{
			id:   'f',
			name: "fast",
			r:    bufio.NewReaderSize(rw, fastProtoReadBufioSize),
			w:    rw,
		}
	}
	lengthSize = int64(binary.Size(uint32(0)))
)

// error
var (
	ErrProtoUnmatch = errors.New("Mismatched protocol")
)

// Version returns the protocol's id and name.
func (f *FastProto) Version() (byte, string) {
	return f.id, f.name
}

// Pack pack socket data packet.
// Note: Make sure to write only once or there will be package contamination!
func (f *FastProto) Pack(p *Packet) error {
	bb := utils.AcquireByteBuffer()
	defer utils.ReleaseByteBuffer(bb)

	// fake size
	err := binary.Write(bb, binary.BigEndian, uint32(0))

	// protocol version
	bb.WriteByte(f.id)

	// transfer pipe
	bb.WriteByte(byte(p.XferPipe().Len()))
	bb.Write(p.XferPipe().Ids())

	prefixLen := bb.Len()

	// header
	err = f.writeHeader(bb, p)
	if err != nil {
		return err
	}

	// body
	err = f.writeBody(bb, p)
	if err != nil {
		return err
	}

	// do transfer pipe
	payload, err := p.XferPipe().OnPack(bb.B[prefixLen:])
	if err != nil {
		return err
	}
	bb.B = append(bb.B[:prefixLen], payload...)

	// set and check packet size
	err = p.SetSize(uint32(bb.Len()))
	if err != nil {
		return err
	}

	// reset real size
	binary.BigEndian.PutUint32(bb.B, p.Size())

	// real write
	_, err = f.w.Write(bb.B)
	if err != nil {
		return err
	}

	return err
}

func (f *FastProto) writeHeader(bb *utils.ByteBuffer, p *Packet) error {
	binary.Write(bb, binary.BigEndian, p.Seq())

	bb.WriteByte(p.Ptype())

	uriBytes := goutil.StringToBytes(p.Uri())
	binary.Write(bb, binary.BigEndian, uint32(len(uriBytes)))
	bb.Write(uriBytes)

	metaBytes := p.Meta().QueryString()
	binary.Write(bb, binary.BigEndian, uint32(len(metaBytes)))
	bb.Write(metaBytes)
	return nil
}

func (f *FastProto) writeBody(bb *utils.ByteBuffer, p *Packet) error {
	bb.WriteByte(p.BodyCodec())
	bodyBytes, err := p.MarshalBody()
	if err != nil {
		return err
	}
	bb.Write(bodyBytes)
	return nil
}

// Unpack unpack socket data packet.
// Note: Concurrent unsafe!
func (f *FastProto) Unpack(p *Packet) error {
	bb := utils.AcquireByteBuffer()
	defer utils.ReleaseByteBuffer(bb)

	// read packet
	err := f.readPacket(bb, p)
	if err != nil {
		return err
	}
	// do transfer pipe
	data, err := p.XferPipe().OnUnpack(bb.B)
	if err != nil {
		return err
	}
	// header
	data = f.readHeader(data, p)
	// body
	return f.readBody(data, p)
}

func (f *FastProto) readPacket(bb *utils.ByteBuffer, p *Packet) error {
	f.rMu.Lock()
	defer f.rMu.Unlock()
	// size
	var size uint32
	err := binary.Read(f.r, binary.BigEndian, &size)
	if err != nil {
		return err
	}
	if err = p.SetSize(size); err != nil {
		return err
	}
	// protocol
	bb.ChangeLen(1024)
	_, err = f.r.Read(bb.B[:1])
	if err != nil {
		return err
	}
	if bb.B[0] != f.id {
		return ErrProtoUnmatch
	}
	// transfer pipe
	_, err = f.r.Read(bb.B[:1])
	if err != nil {
		return err
	}
	var xferLen = bb.B[0]
	if xferLen > 0 {
		_, err = f.r.Read(bb.B[:xferLen])
		if err != nil {
			return err
		}
		err = p.XferPipe().Append(bb.B[:xferLen]...)
		if err != nil {
			return err
		}
	}
	// read last all
	var lastLen = int(size) - 4 - 1 - 1 - int(xferLen)
	bb.ChangeLen(lastLen)
	_, err = io.ReadFull(f.r, bb.B)
	return err
}

func (f *FastProto) readHeader(data []byte, p *Packet) []byte {
	// seq
	p.SetSeq(binary.BigEndian.Uint64(data))
	data = data[8:]
	// type
	p.SetPtype(data[0])
	data = data[1:]
	// uri
	uriLen := binary.BigEndian.Uint32(data)
	data = data[4:]
	p.SetUri(string(data[:uriLen]))
	data = data[uriLen:]
	// meta
	metaLen := binary.BigEndian.Uint32(data)
	data = data[4:]
	p.Meta().ParseBytes(data[:metaLen])
	data = data[metaLen:]
	return data
}

func (f *FastProto) readBody(data []byte, p *Packet) error {
	p.SetBodyCodec(data[0])
	return p.UnmarshalNewBody(data[1:])
}
