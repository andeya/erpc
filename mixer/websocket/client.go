// Websocket is an extension package that makes the Teleport framework compatible
// with websocket protocol as specified in RFC 6455.
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
//
package websocket

import (
	"net"
	"path"
	"strings"

	tp "github.com/henrylee2cn/teleport"
	ws "github.com/henrylee2cn/teleport/mixer/websocket/websocket"
	"github.com/henrylee2cn/teleport/socket"
)

// NewDialPlugin creates a websocket plugin for client.
func NewDialPlugin(pattern string) tp.Plugin {
	pattern = path.Join("/", strings.TrimRight(pattern, "/"))
	if pattern == "/" {
		pattern = ""
	}
	return &clientPlugin{pattern}
}

type clientPlugin struct {
	pattern string
}

var (
	_ tp.PostDialPlugin = new(clientPlugin)
)

func (*clientPlugin) Name() string {
	return "websocket"
}

func (c *clientPlugin) PostDial(sess tp.PreSession) *tp.Rerror {
	var location, origin string
	if sess.Peer().TlsConfig() == nil {
		location = "ws://" + sess.RemoteAddr().String() + c.pattern
		origin = "ws://" + sess.LocalAddr().String() + c.pattern
	} else {
		location = "wss://" + sess.RemoteAddr().String() + c.pattern
		origin = "wss://" + sess.LocalAddr().String() + c.pattern
	}
	cfg, err := ws.NewConfig(location, origin)
	if err != nil {
		return tp.NewRerror(tp.CodeDialFailed, "upgrade to websocket failed", err.Error())
	}
	var rerr *tp.Rerror
	sess.ModifySocket(func(conn net.Conn) (net.Conn, socket.ProtoFunc) {
		conn, err := ws.NewClient(cfg, conn)
		if err != nil {
			rerr = tp.NewRerror(tp.CodeDialFailed, "upgrade to websocket failed", err.Error())
			return nil, nil
		}
		return conn, NewWsProtoFunc(sess.GetProtoFunc())
	})
	return rerr
}
