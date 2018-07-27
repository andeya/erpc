// Package pbproto is implemented PROTOBUF socket communication protocol.
//  Packet data format: {length bytes}{xfer_pipe length byte}{xfer_pipe bytes}{JSON bytes}
//
// Copyright 2018 HenryLee. All Rights Reserved.
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
package pbproto

import (
	"bufio"
	"encoding/binary"
	"io"
	"sync"

	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/socket"
	"github.com/henrylee2cn/teleport/utils"
	"github.com/henrylee2cn/tp-ext/proto-pbproto/pb"
)

// NewPbProtoFunc is creation function of PROTOBUF socket protocol.
var NewPbProtoFunc = func(rw io.ReadWriter) socket.Proto {
	var (
		readBufioSize             int
		readBufferSize, isDefault = socket.ReadBuffer()
	)
	if isDefault {
		readBufioSize = 1024 * 4
	} else if readBufferSize == 0 {
		readBufioSize = 1024 * 35
	} else {
		readBufioSize = readBufferSize / 2
	}
	return &pbproto{
		id:   'p',
		name: "protobuf",
		r:    bufio.NewReaderSize(rw, readBufioSize),
		w:    rw,
	}
}

type pbproto struct {
	id   byte
	name string
	r    *bufio.Reader
	w    io.Writer
	rMu  sync.Mutex
}

// Version returns the protocol's id and name.
func (pp *pbproto) Version() (byte, string) {
	return pp.id, pp.name
}

// Pack writes the Packet into the connection.
// Note: Make sure to write only once or there will be package contamination!
func (pp *pbproto) Pack(p *socket.Packet) error {
	// marshal body
	bodyBytes, err := p.MarshalBody()
	if err != nil {
		return err
	}

	b, err := codec.ProtoMarshal(&pb.Payload{
		Seq:       p.Seq(),
		Ptype:     int32(p.Ptype()),
		Uri:       p.Uri(),
		Meta:      p.Meta().QueryString(),
		BodyCodec: int32(p.BodyCodec()),
		Body:      bodyBytes,
	})
	if err != nil {
		return err
	}

	// do transfer pipe
	b, err = p.XferPipe().OnPack(b)
	if err != nil {
		return err
	}
	xferPipeLen := p.XferPipe().Len()

	// set size
	p.SetSize(uint32(1 + xferPipeLen + len(b)))

	// pack
	var all = make([]byte, p.Size()+4)
	binary.BigEndian.PutUint32(all, p.Size())
	all[4] = byte(xferPipeLen)
	copy(all[4+1:], p.XferPipe().Ids())
	copy(all[4+1+xferPipeLen:], b)
	_, err = pp.w.Write(all)
	return err
}

// Unpack reads bytes from the connection to the Packet.
// Note: Concurrent unsafe!
func (pp *pbproto) Unpack(p *socket.Packet) error {
	pp.rMu.Lock()
	defer pp.rMu.Unlock()
	var size uint32
	err := binary.Read(pp.r, binary.BigEndian, &size)
	if err != nil {
		return err
	}
	if err = p.SetSize(size); err != nil {
		return err
	}
	if p.Size() == 0 {
		return nil
	}

	bb := utils.AcquireByteBuffer()
	defer utils.ReleaseByteBuffer(bb)
	bb.ChangeLen(int(p.Size()))
	_, err = io.ReadFull(pp.r, bb.B)
	if err != nil {
		return err
	}

	// transfer pipe
	var xferLen = bb.B[0]
	bb.B = bb.B[1:]
	if xferLen > 0 {
		err = p.XferPipe().Append(bb.B[:xferLen]...)
		if err != nil {
			return err
		}
		bb.B = bb.B[xferLen:]
		// do transfer pipe
		bb.B, err = p.XferPipe().OnUnpack(bb.B)
		if err != nil {
			return err
		}
	}

	s := &pb.Payload{}
	err = codec.ProtoUnmarshal(bb.B, s)
	if err != nil {
		return err
	}

	// read other
	p.SetSeq(s.Seq)
	p.SetPtype(byte(s.Ptype))
	p.SetUri(s.Uri)
	p.Meta().ParseBytes(s.Meta)

	// read body
	p.SetBodyCodec(byte(s.BodyCodec))
	err = p.UnmarshalBody(s.Body)
	return err
}
