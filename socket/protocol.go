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
		// Pack writes the Packet into the connection.
		// Note: Make sure to write only once or there will be package contamination!
		Pack(*Packet) error
		// Unpack reads bytes from the connection to the Packet.
		// Note: Concurrent unsafe!
		Unpack(*Packet) error
	}
	// ProtoFunc function used to create a custom Proto interface.
	ProtoFunc func(io.ReadWriter) Proto
)

// default builder of socket communication protocol.
var defaultProtoFunc = NewFastProtoFunc

// DefaultProtoFunc gets the default builder of socket communication protocol
func DefaultProtoFunc() ProtoFunc {
	return defaultProtoFunc
}

// SetDefaultProtoFunc sets the default builder of socket communication protocol
func SetDefaultProtoFunc(protoFunc ProtoFunc) {
	defaultProtoFunc = protoFunc
}

// default protocol

// fastProto fast socket communication protocol.
type fastProto struct {
	id   byte
	name string
	r    io.Reader
	w    io.Writer
	rMu  sync.Mutex
}

// NewFastProtoFunc is creation function of fast socket protocol.
// NOTE: it is the default protocol.
var NewFastProtoFunc = func(rw io.ReadWriter) Proto {
	var (
		fastProtoReadBufioSize    int
		readBufferSize, isDefault = ReadBuffer()
	)
	if isDefault {
		fastProtoReadBufioSize = 1024 * 4
	} else if readBufferSize == 0 {
		fastProtoReadBufioSize = 1024 * 35
	} else {
		fastProtoReadBufioSize = readBufferSize / 2
	}
	return &fastProto{
		id:   'f',
		name: "fast",
		r:    bufio.NewReaderSize(rw, fastProtoReadBufioSize),
		w:    rw,
	}
}

// Version returns the protocol's id and name.
func (f *fastProto) Version() (byte, string) {
	return f.id, f.name
}

// Pack writes the Packet into the connection.
// Note: Make sure to write only once or there will be package contamination!
func (f *fastProto) Pack(p *Packet) error {
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

func (f *fastProto) writeHeader(bb *utils.ByteBuffer, p *Packet) error {
	seqBytes := goutil.StringToBytes(p.Seq())
	binary.Write(bb, binary.BigEndian, uint32(len(seqBytes)))
	bb.Write(seqBytes)

	bb.WriteByte(p.Ptype())

	uriBytes := goutil.StringToBytes(p.Uri())
	binary.Write(bb, binary.BigEndian, uint32(len(uriBytes)))
	bb.Write(uriBytes)

	metaBytes := p.Meta().QueryString()
	binary.Write(bb, binary.BigEndian, uint32(len(metaBytes)))
	bb.Write(metaBytes)
	return nil
}

func (f *fastProto) writeBody(bb *utils.ByteBuffer, p *Packet) error {
	bb.WriteByte(p.BodyCodec())
	bodyBytes, err := p.MarshalBody()
	if err != nil {
		return err
	}
	bb.Write(bodyBytes)
	return nil
}

// Unpack reads bytes from the connection to the Packet.
// Note: Concurrent unsafe!
func (f *fastProto) Unpack(p *Packet) error {
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

var errProtoUnmatch = errors.New("mismatched protocol")

func (f *fastProto) readPacket(bb *utils.ByteBuffer, p *Packet) error {
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
	_, err = io.ReadFull(f.r, bb.B[:1])
	if err != nil {
		return err
	}
	if bb.B[0] != f.id {
		return errProtoUnmatch
	}
	// transfer pipe
	_, err = io.ReadFull(f.r, bb.B[:1])
	if err != nil {
		return err
	}
	var xferLen = bb.B[0]
	if xferLen > 0 {
		_, err = io.ReadFull(f.r, bb.B[:xferLen])
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

func (f *fastProto) readHeader(data []byte, p *Packet) []byte {
	// seq
	seqLen := binary.BigEndian.Uint32(data)
	data = data[4:]
	p.SetSeq(string(data[:seqLen]))
	data = data[seqLen:]
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

func (f *fastProto) readBody(data []byte, p *Packet) error {
	p.SetBodyCodec(data[0])
	return p.UnmarshalBody(data[1:])
}
