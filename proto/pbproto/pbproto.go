// Package pbproto is implemented PROTOBUF socket communication protocol.
//  Message data format: {length bytes}{xfer_pipe length byte}{xfer_pipe bytes}{JSON bytes}
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
	"encoding/binary"
	"io"
	"sync"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/proto/pbproto/pb"
	"github.com/henrylee2cn/teleport/utils"
)

// NewPbProtoFunc is creation function of PROTOBUF socket protocol.
func NewPbProtoFunc() tp.ProtoFunc {
	return func(rw tp.IOWithReadBuffer) tp.Proto {
		return &pbproto{
			id:   'p',
			name: "protobuf",
			rw:   rw,
		}
	}
}

type pbproto struct {
	rw   tp.IOWithReadBuffer
	rMu  sync.Mutex
	name string
	id   byte
}

// Version returns the protocol's id and name.
func (pp *pbproto) Version() (byte, string) {
	return pp.id, pp.name
}

// Pack writes the Message into the connection.
// NOTE: Make sure to write only once or there will be package contamination!
func (pp *pbproto) Pack(m tp.Message) error {
	// marshal body
	bodyBytes, err := m.MarshalBody()
	if err != nil {
		return err
	}

	b, err := codec.ProtoMarshal(&pb.Payload{
		Seq:           m.Seq(),
		Mtype:         int32(m.Mtype()),
		ServiceMethod: m.ServiceMethod(),
		Status:        m.Status(true).EncodeQuery(),
		Meta:          m.Meta().QueryString(),
		BodyCodec:     int32(m.BodyCodec()),
		Body:          bodyBytes,
	})
	if err != nil {
		return err
	}

	// do transfer pipe
	b, err = m.XferPipe().OnPack(b)
	if err != nil {
		return err
	}
	xferPipeLen := m.XferPipe().Len()

	// set size
	m.SetSize(uint32(1 + xferPipeLen + len(b)))

	// pack
	var all = make([]byte, m.Size()+4)
	binary.BigEndian.PutUint32(all, m.Size())
	all[4] = byte(xferPipeLen)
	copy(all[4+1:], m.XferPipe().IDs())
	copy(all[4+1+xferPipeLen:], b)
	_, err = pp.rw.Write(all)
	return err
}

// Unpack reads bytes from the connection to the Message.
func (pp *pbproto) Unpack(m tp.Message) error {
	pp.rMu.Lock()
	defer pp.rMu.Unlock()
	var size uint32
	err := binary.Read(pp.rw, binary.BigEndian, &size)
	if err != nil {
		return err
	}
	if err = m.SetSize(size); err != nil {
		return err
	}
	if m.Size() == 0 {
		return nil
	}

	bb := utils.AcquireByteBuffer()
	defer utils.ReleaseByteBuffer(bb)
	bb.ChangeLen(int(m.Size()))
	_, err = io.ReadFull(pp.rw, bb.B)
	if err != nil {
		return err
	}

	// transfer pipe
	var xferLen = bb.B[0]
	bb.B = bb.B[1:]
	if xferLen > 0 {
		err = m.XferPipe().Append(bb.B[:xferLen]...)
		if err != nil {
			return err
		}
		bb.B = bb.B[xferLen:]
		// do transfer pipe
		bb.B, err = m.XferPipe().OnUnpack(bb.B)
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
	m.SetSeq(s.Seq)
	m.SetMtype(byte(s.Mtype))
	m.SetServiceMethod(s.ServiceMethod)
	m.Status(true).DecodeQuery(s.Status)
	m.Meta().ParseBytes(s.Meta)

	// read body
	m.SetBodyCodec(byte(s.BodyCodec))
	err = m.UnmarshalBody(s.Body)
	return err
}
