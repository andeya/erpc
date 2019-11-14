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

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/mixer/websocket/jsonSubProto"
	ws "github.com/henrylee2cn/erpc/v6/mixer/websocket/websocket"
	"github.com/henrylee2cn/erpc/v6/socket"
	"github.com/henrylee2cn/erpc/v6/utils"
)

var defaultProto = jsonSubProto.NewJSONSubProtoFunc()

// NewWsProtoFunc wraps a protocol to a new websocket protocol.
func NewWsProtoFunc(subProto ...erpc.ProtoFunc) erpc.ProtoFunc {
	return func(rw erpc.IOWithReadBuffer) socket.Proto {
		// When called, the lock of the external socket.Socket is already locked,
		// so it is concurrent security.
		connIface := rw.(socket.UnsafeSocket).RawLocked()
		conn, ok := connIface.(*ws.Conn)
		if !ok {
			if len(subProto) > 0 {
				return subProto[0](rw)
			}
			return defaultProto(rw)
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

// Pack writes the Message into the connection.
// NOTE: Make sure to write only once or there will be package contamination!
func (w *wsProto) Pack(m erpc.Message) error {
	w.subConn.w.Reset()
	err := w.subProto.Pack(m)
	if err != nil {
		return err
	}
	return ws.Message.Send(w.conn, w.subConn.w.Bytes())
}

// Unpack reads bytes from the connection to the Message.
// NOTE: Concurrent unsafe!
func (w *wsProto) Unpack(m erpc.Message) error {
	err := ws.Message.Receive(w.conn, w.subConn.rBytes)
	if err != nil {
		return err
	}
	w.subConn.r = bytes.NewBuffer(*w.subConn.rBytes)
	return w.subProto.Unpack(m)
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
