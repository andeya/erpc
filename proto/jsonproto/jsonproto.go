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
	"fmt"
	"io"
	"sync"

	"github.com/tidwall/gjson"

	"github.com/henrylee2cn/goutil"
	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/utils"
)

// NewJsonProtoFunc is creation function of JSON socket protocol.
//  Message data format: {length bytes}{xfer_pipe length byte}{xfer_pipe bytes}{JSON bytes}
//  Message data demo: `830{"seq":%q,"mtype":%d,"uri":%q,"meta":%q,"body_codec":%d,"body":"%s"}`
var NewJsonProtoFunc = func(rw io.ReadWriter) tp.Proto {
	return &jsonproto{
		id:   'j',
		name: "json",
		rw:   rw,
	}
}

type jsonproto struct {
	id   byte
	name string
	rw   io.ReadWriter
	rMu  sync.Mutex
}

// Version returns the protocol's id and name.
func (j *jsonproto) Version() (byte, string) {
	return j.id, j.name
}

const format = `{"seq":%q,"mtype":%d,"uri":%q,"meta":%q,"body_codec":%d,"body":"%s"}`

// Pack writes the Message into the connection.
// Note: Make sure to write only once or there will be package contamination!
func (j *jsonproto) Pack(m *tp.Message) error {
	// marshal body
	bodyBytes, err := m.MarshalBody()
	if err != nil {
		return err
	}

	// marshal whole
	var s = fmt.Sprintf(format,
		m.Seq(),
		m.Mtype(),
		m.Uri(),
		m.Meta().QueryString(),
		m.BodyCodec(),
		bytes.Replace(bodyBytes, []byte{'"'}, []byte{'\\', '"'}, -1),
	)

	// do transfer pipe
	b, err := m.XferPipe().OnPack(goutil.StringToBytes(s))
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
	copy(all[4+1:], m.XferPipe().Ids())
	copy(all[4+1+xferPipeLen:], b)
	_, err = j.rw.Write(all)
	return err
}

// Unpack reads bytes from the connection to the Message.
// Note: Concurrent unsafe!
func (j *jsonproto) Unpack(m *tp.Message) error {
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
	m.SetSeq(gjson.Get(s, "seq").String())
	m.SetMtype(byte(gjson.Get(s, "mtype").Int()))
	m.SetUri(gjson.Get(s, "uri").String())
	meta := gjson.Get(s, "meta").String()
	m.Meta().ParseBytes(goutil.StringToBytes(meta))

	// read body
	m.SetBodyCodec(byte(gjson.Get(s, "body_codec").Int()))
	body := gjson.Get(s, "body").String()
	err = m.UnmarshalBody(goutil.StringToBytes(body))
	return err
}
