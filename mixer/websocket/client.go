// Package websocket is an extension package that makes the eRPC framework compatible
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

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/mixer/websocket/jsonSubProto"
	"github.com/henrylee2cn/erpc/v6/mixer/websocket/pbSubProto"
	ws "github.com/henrylee2cn/erpc/v6/mixer/websocket/websocket"
)

// Client a websocket client
type Client struct {
	erpc.Peer
}

// NewClient creates a websocket client.
func NewClient(rootPath string, cfg erpc.PeerConfig, globalLeftPlugin ...erpc.Plugin) *Client {
	globalLeftPlugin = append([]erpc.Plugin{NewDialPlugin(rootPath)}, globalLeftPlugin...)
	peer := erpc.NewPeer(cfg, globalLeftPlugin...)
	return &Client{
		Peer: peer,
	}
}

// DialJSON connects with the JSON protocol.
func (c *Client) DialJSON(addr string) (erpc.Session, *erpc.Status) {
	return c.Dial(addr, jsonSubProto.NewJSONSubProtoFunc())
}

// DialProtobuf connects with the Protobuf protocol.
func (c *Client) DialProtobuf(addr string) (erpc.Session, *erpc.Status) {
	return c.Dial(addr, pbSubProto.NewPbSubProtoFunc())
}

// Dial connects with the peer of the destination address.
func (c *Client) Dial(addr string, protoFunc ...erpc.ProtoFunc) (erpc.Session, *erpc.Status) {
	if len(protoFunc) == 0 {
		return c.Peer.Dial(addr, defaultProto)
	}
	return c.Peer.Dial(addr, protoFunc...)
}

// NewDialPlugin creates a websocket plugin for client.
func NewDialPlugin(rootPath string) erpc.Plugin {
	return &clientPlugin{fixRootPath(rootPath)}
}

func fixRootPath(rootPath string) string {
	rootPath = path.Join("/", strings.TrimRight(rootPath, "/"))
	return rootPath
}

type clientPlugin struct {
	rootPath string
}

var (
	_ erpc.PostDialPlugin = new(clientPlugin)
)

func (*clientPlugin) Name() string {
	return "websocket"
}

func (c *clientPlugin) PostDial(sess erpc.PreSession, isRedial bool) (stat *erpc.Status) {
	var location, origin string
	if sess.Peer().TLSConfig() == nil {
		location = "ws://" + sess.RemoteAddr().String() + c.rootPath
		origin = "ws://" + sess.LocalAddr().String() + c.rootPath
	} else {
		location = "wss://" + sess.RemoteAddr().String() + c.rootPath
		origin = "wss://" + sess.LocalAddr().String() + c.rootPath
	}
	cfg, err := ws.NewConfig(location, origin)
	if err != nil {
		return erpc.NewStatus(erpc.CodeDialFailed, "upgrade to websocket failed", err.Error())
	}
	sess.ModifySocket(func(conn net.Conn) (net.Conn, erpc.ProtoFunc) {
		conn, err := ws.NewClient(cfg, conn)
		if err != nil {
			stat = erpc.NewStatus(erpc.CodeDialFailed, "upgrade to websocket failed", err.Error())
			return nil, nil
		}
		if isRedial {
			return conn, sess.GetProtoFunc()
		}
		return conn, NewWsProtoFunc(sess.GetProtoFunc())
	})
	return stat
}
