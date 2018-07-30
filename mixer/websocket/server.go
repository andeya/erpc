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
	"net/http"
	"net/url"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/mixer/websocket/jsonSubProto"
	"github.com/henrylee2cn/teleport/mixer/websocket/pbSubProto"
	ws "github.com/henrylee2cn/teleport/mixer/websocket/websocket"
	"github.com/henrylee2cn/teleport/socket"
)

// NewJsonServeHandler creates a websocket json handler.
func NewJsonServeHandler(peer tp.Peer, handshake func(*ws.Config, *http.Request) error) http.Handler {
	return NewServeHandler(peer, handshake, jsonSubProto.NewJsonSubProtoFunc)
}

// NewPbServeHandler creates a websocket protobuf handler.
func NewPbServeHandler(peer tp.Peer, handshake func(*ws.Config, *http.Request) error) http.Handler {
	return NewServeHandler(peer, handshake, pbSubProto.NewPbSubProtoFunc)
}

// NewServeHandler creates a websocket handler.
func NewServeHandler(peer tp.Peer, handshake func(*ws.Config, *http.Request) error, protoFunc ...socket.ProtoFunc) http.Handler {
	w := &serverHandler{
		peer:      peer,
		Server:    new(ws.Server),
		protoFunc: NewWsProtoFunc(protoFunc...),
	}
	var scheme string
	if peer.TlsConfig() == nil {
		scheme = "ws"
	} else {
		scheme = "wss"
	}
	if handshake != nil {
		w.Server.Handshake = func(cfg *ws.Config, r *http.Request) error {
			cfg.Origin = &url.URL{
				Host:   r.RemoteAddr,
				Scheme: scheme,
			}
			return handshake(cfg, r)
		}
	} else {
		w.Server.Handshake = func(cfg *ws.Config, r *http.Request) error {
			cfg.Origin = &url.URL{
				Host:   r.RemoteAddr,
				Scheme: scheme,
			}
			return nil
		}
	}
	w.Server.Handler = w.handler
	w.Server.Config = ws.Config{
		TlsConfig: peer.TlsConfig(),
	}
	return w
}

type serverHandler struct {
	peer      tp.Peer
	protoFunc socket.ProtoFunc
	*ws.Server
}

func (w *serverHandler) handler(conn *ws.Conn) {
	sess, err := w.peer.ServeConn(conn, w.protoFunc)
	if err != nil {
		tp.Errorf("serverHandler: %v", err)
	}
	<-sess.CloseNotify()
}
