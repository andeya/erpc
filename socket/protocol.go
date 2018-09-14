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
	"encoding/binary"
	"errors"
	"io"
	"sync"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/teleport/utils"
)

type (
	// Proto pack/unpack protocol scheme of socket message.
	Proto interface {
		// Version returns the protocol's id and name.
		Version() (byte, string)
		// Pack writes the Message into the connection.
		// Note: Make sure to write only once or there will be package contamination!
		Pack(*Message) error
		// Unpack reads bytes from the connection to the Message.
		// Note: Concurrent unsafe!
		Unpack(*Message) error
	}
	// ProtoFunc function used to create a custom Proto interface.
	ProtoFunc func(io.ReadWriter) Proto
)

// default builder of socket communication protocol.
var defaultProtoFunc = NewRawProtoFunc

// DefaultProtoFunc gets the default builder of socket communication protocol
func DefaultProtoFunc() ProtoFunc {
	return defaultProtoFunc
}

// SetDefaultProtoFunc sets the default builder of socket communication protocol
func SetDefaultProtoFunc(protoFunc ProtoFunc) {
	defaultProtoFunc = protoFunc
}

// default protocol

/*
# raw protocol format(Big Endian):

{4 bytes message length}
{1 byte protocol version}
{1 byte transfer pipe length}
{transfer pipe IDs}
# The following is handled data by transfer pipe
{2 bytes sequence length}
{sequence}
{1 byte message type} # e.g. CALL:1; REPLY:2; PUSH:3
{2 bytes URI length}
{URI}
{2 bytes metadata length}
{metadata(urlencoded)}
{1 byte body codec id}
{body}
*/

// rawProto fast socket communication protocol.
type rawProto struct {
	id   byte
	name string
	r    io.Reader
	w    io.Writer
	rMu  sync.Mutex
}

// NewRawProtoFunc is creation function of fast socket protocol.
// NOTE: it is the default protocol.
var NewRawProtoFunc = func(rw io.ReadWriter) Proto {
	return &rawProto{
		id:   'r',
		name: "raw",
		r:    rw,
		w:    rw,
	}
}

// Version returns the protocol's id and name.
func (r *rawProto) Version() (byte, string) {
	return r.id, r.name
}

// Pack writes the Message into the connection.
// Note: Make sure to write only once or there will be package contamination!
func (r *rawProto) Pack(m *Message) error {
	bb := utils.AcquireByteBuffer()
	defer utils.ReleaseByteBuffer(bb)

	// fake size
	err := binary.Write(bb, binary.BigEndian, uint32(0))

	// protocol version
	bb.WriteByte(r.id)

	// transfer pipe
	bb.WriteByte(byte(m.XferPipe().Len()))
	bb.Write(m.XferPipe().Ids())

	prefixLen := bb.Len()

	// header
	err = r.writeHeader(bb, m)
	if err != nil {
		return err
	}

	// body
	err = r.writeBody(bb, m)
	if err != nil {
		return err
	}

	// do transfer pipe
	payload, err := m.XferPipe().OnPack(bb.B[prefixLen:])
	if err != nil {
		return err
	}
	bb.B = append(bb.B[:prefixLen], payload...)

	// set and check message size
	err = m.SetSize(uint32(bb.Len()))
	if err != nil {
		return err
	}

	// reset real size
	binary.BigEndian.PutUint32(bb.B, m.Size())

	// real write
	_, err = r.w.Write(bb.B)
	if err != nil {
		return err
	}

	return err
}

func (r *rawProto) writeHeader(bb *utils.ByteBuffer, m *Message) error {
	seqBytes := goutil.StringToBytes(m.Seq())
	binary.Write(bb, binary.BigEndian, uint16(len(seqBytes)))
	bb.Write(seqBytes)

	bb.WriteByte(m.Mtype())

	uriBytes := goutil.StringToBytes(m.Uri())
	binary.Write(bb, binary.BigEndian, uint16(len(uriBytes)))
	bb.Write(uriBytes)

	metaBytes := m.Meta().QueryString()
	binary.Write(bb, binary.BigEndian, uint16(len(metaBytes)))
	bb.Write(metaBytes)
	return nil
}

func (r *rawProto) writeBody(bb *utils.ByteBuffer, m *Message) error {
	bb.WriteByte(m.BodyCodec())
	bodyBytes, err := m.MarshalBody()
	if err != nil {
		return err
	}
	bb.Write(bodyBytes)
	return nil
}

// Unpack reads bytes from the connection to the Message.
// Note: Concurrent unsafe!
func (r *rawProto) Unpack(m *Message) error {
	bb := utils.AcquireByteBuffer()
	defer utils.ReleaseByteBuffer(bb)

	// read message
	err := r.readMessage(bb, m)
	if err != nil {
		return err
	}
	// do transfer pipe
	data, err := m.XferPipe().OnUnpack(bb.B)
	if err != nil {
		return err
	}
	// header
	data = r.readHeader(data, m)
	// body
	return r.readBody(data, m)
}

var errProtoUnmatch = errors.New("mismatched protocol")

func (r *rawProto) readMessage(bb *utils.ByteBuffer, m *Message) error {
	r.rMu.Lock()
	defer r.rMu.Unlock()
	// size
	var size uint32
	err := binary.Read(r.r, binary.BigEndian, &size)
	if err != nil {
		return err
	}
	if err = m.SetSize(size); err != nil {
		return err
	}
	// protocol
	bb.ChangeLen(1024)
	_, err = io.ReadFull(r.r, bb.B[:1])
	if err != nil {
		return err
	}
	if bb.B[0] != r.id {
		return errProtoUnmatch
	}
	// transfer pipe
	_, err = io.ReadFull(r.r, bb.B[:1])
	if err != nil {
		return err
	}
	var xferLen = bb.B[0]
	if xferLen > 0 {
		_, err = io.ReadFull(r.r, bb.B[:xferLen])
		if err != nil {
			return err
		}
		err = m.XferPipe().Append(bb.B[:xferLen]...)
		if err != nil {
			return err
		}
	}
	// read last all
	var lastLen = int(size) - 4 - 1 - 1 - int(xferLen)
	bb.ChangeLen(lastLen)
	_, err = io.ReadFull(r.r, bb.B)
	return err
}

func (r *rawProto) readHeader(data []byte, m *Message) []byte {
	// seq
	seqLen := binary.BigEndian.Uint16(data)
	data = data[2:]
	m.SetSeq(string(data[:seqLen]))
	data = data[seqLen:]
	// type
	m.SetMtype(data[0])
	data = data[1:]
	// uri
	uriLen := binary.BigEndian.Uint16(data)
	data = data[2:]
	m.SetUri(string(data[:uriLen]))
	data = data[uriLen:]
	// meta
	metaLen := binary.BigEndian.Uint16(data)
	data = data[2:]
	m.Meta().ParseBytes(data[:metaLen])
	data = data[metaLen:]
	return data
}

func (r *rawProto) readBody(data []byte, m *Message) error {
	m.SetBodyCodec(data[0])
	return m.UnmarshalBody(data[1:])
}
