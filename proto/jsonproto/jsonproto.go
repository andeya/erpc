// Package jsonproto is implemented JSON socket communication protocol.
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
package jsonproto

import (
	"bytes"
	"encoding/binary"
	"io"
	"strconv"
	"sync"

	"github.com/tidwall/gjson"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/utils"
	"github.com/henrylee2cn/goutil"
)

// NewJSONProtoFunc is creation function of JSON socket protocol.
//  Message data format: {length bytes}{xfer_pipe length byte}{xfer_pipe bytes}{JSON bytes}
//  Message data demo: `830{"seq":%q,"mtype":%d,"serviceMethod":%q,"meta":%q,"bodyCodec":%d,"body":"%s"}`
func NewJSONProtoFunc() erpc.ProtoFunc {
	return func(rw erpc.IOWithReadBuffer) erpc.Proto {
		return &jsonproto{
			id:   'j',
			name: "json",
			rw:   rw,
		}
	}
}

type jsonproto struct {
	rw   erpc.IOWithReadBuffer
	rMu  sync.Mutex
	name string
	id   byte
}

// Version returns the protocol's id and name.
func (j *jsonproto) Version() (byte, string) {
	return j.id, j.name
}

// const format = `{"seq":%d,"mtype":%d,"serviceMethod":%q,"status":%q,"meta":%q,"bodyCodec":%d,"body":"%s"}`

var (
	msg1 = []byte(`{"seq":`)
	msg2 = []byte(`,"mtype":`)
	msg3 = []byte(`,"serviceMethod":`)
	msg4 = []byte(`,"status":`)
	msg5 = []byte(`,"meta":`)
	msg6 = []byte(`,"bodyCodec":`)
	msg7 = []byte(`,"body":"`)
	msg8 = []byte(`"}`)
)

// Pack writes the Message into the connection.
// NOTE: Make sure to write only once or there will be package contamination!
func (j *jsonproto) Pack(m erpc.Message) error {
	// marshal body
	bodyBytes, err := m.MarshalBody()
	if err != nil {
		return err
	}

	// marshal whole
	bb := utils.AcquireByteBuffer()
	defer utils.ReleaseByteBuffer(bb)
	bb.Write(msg1)
	bb.WriteString(strconv.FormatInt(int64(m.Seq()), 10))
	bb.Write(msg2)
	bb.WriteString(strconv.FormatInt(int64(m.Mtype()), 10))
	bb.Write(msg3)
	bb.WriteString(strconv.Quote(m.ServiceMethod()))
	bb.Write(msg4)
	bb.WriteString(strconv.Quote(m.Status(true).QueryString()))
	bb.Write(msg5)
	bb.WriteString(strconv.Quote(goutil.BytesToString(m.Meta().QueryString())))
	bb.Write(msg6)
	bb.WriteString(strconv.FormatInt(int64(m.BodyCodec()), 10))
	bb.Write(msg7)
	bb.Write(bytes.Replace(bodyBytes, []byte{'"'}, []byte{'\\', '"'}, -1))
	bb.Write(msg8)

	// do transfer pipe
	b, err := m.XferPipe().OnPack(bb.B)
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
	_, err = j.rw.Write(all)
	return err
}

// Unpack reads bytes from the connection to the Message.
func (j *jsonproto) Unpack(m erpc.Message) error {
	j.rMu.Lock()
	defer j.rMu.Unlock()
	var size uint32
	err := binary.Read(j.rw, binary.BigEndian, &size)
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
	_, err = io.ReadFull(j.rw, bb.B)
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

	s := string(bb.B)

	// read other
	m.SetSeq(int32(gjson.Get(s, "seq").Int()))
	m.SetMtype(byte(gjson.Get(s, "mtype").Int()))
	m.SetServiceMethod(gjson.Get(s, "serviceMethod").String())
	stat := gjson.Get(s, "status").String()
	m.Status(true).DecodeQuery(goutil.StringToBytes(stat))
	meta := gjson.Get(s, "meta").String()
	m.Meta().ParseBytes(goutil.StringToBytes(meta))

	// read body
	m.SetBodyCodec(byte(gjson.Get(s, "bodyCodec").Int()))
	body := gjson.Get(s, "body").String()
	err = m.UnmarshalBody(goutil.StringToBytes(body))
	return err
}
