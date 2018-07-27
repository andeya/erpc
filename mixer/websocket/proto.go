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

package websocket

import (
	"bytes"
	"io"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/mixer/websocket/jsonSubProto"
	ws "github.com/henrylee2cn/teleport/mixer/websocket/websocket"
	"github.com/henrylee2cn/teleport/socket"
	"github.com/henrylee2cn/teleport/utils"
)

var defaultProto = jsonSubProto.NewJsonSubProtoFunc

// NewWsProtoFunc wraps a protocol to a new websocket protocol.
func NewWsProtoFunc(subProto ...socket.ProtoFunc) socket.ProtoFunc {
	return func(rw io.ReadWriter) socket.Proto {
		conn, ok := rw.(*ws.Conn)
		if !ok {
			tp.Warnf("connection does not support websocket protocol")
			if len(subProto) > 0 {
				return subProto[0](rw)
			} else {
				return socket.DefaultProtoFunc()(rw)
			}
		}
		subConn := newVirtualConn()
		p := &wsProto{
			id:      'w',
			name:    "websocket",
			conn:    conn,
			subConn: subConn,
		}
		if len(subProto) > 0 {
			p.subProto = subProto[0](subConn)
		} else {
			p.subProto = defaultProto(subConn)
		}
		return p
	}
}

type wsProto struct {
	id       byte
	name     string
	conn     *ws.Conn
	subProto socket.Proto
	subConn  *virtualConn
}

// Version returns the protocol's id and name.
func (w *wsProto) Version() (byte, string) {
	return w.id, w.name
}

// Pack writes the Packet into the connection.
// Note: Make sure to write only once or there will be package contamination!
func (w *wsProto) Pack(p *socket.Packet) error {
	w.subConn.w.Reset()
	err := w.subProto.Pack(p)
	if err != nil {
		return err
	}
	return ws.Message.Send(w.conn, w.subConn.w.Bytes())
}

// Unpack reads bytes from the connection to the Packet.
// Note: Concurrent unsafe!
func (w *wsProto) Unpack(p *socket.Packet) error {
	err := ws.Message.Receive(w.conn, w.subConn.rBytes)
	if err != nil {
		return err
	}
	w.subConn.r = bytes.NewBuffer(*w.subConn.rBytes)
	return w.subProto.Unpack(p)
}

func newVirtualConn() *virtualConn {
	buf := new([]byte)
	return &virtualConn{
		rBytes: buf,
		r:      bytes.NewBuffer(*buf),
		w:      utils.AcquireByteBuffer(),
	}
}

type virtualConn struct {
	rBytes *[]byte
	r      *bytes.Buffer
	w      *utils.ByteBuffer
}

func (v *virtualConn) Read(p []byte) (int, error) {
	return v.r.Read(p)
}

func (v *virtualConn) Write(p []byte) (int, error) {
	return v.w.Write(p)
}
