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
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/mixer/websocket/jsonSubProto"
	"github.com/henrylee2cn/teleport/mixer/websocket/pbSubProto"
	ws "github.com/henrylee2cn/teleport/mixer/websocket/websocket"
)

// Server a websocket server
type Server struct {
	tp.Peer
	cfg       tp.PeerConfig
	serveMux  *http.ServeMux
	server    *http.Server
	pattern   string
	lis       net.Listener
	handshake func(*ws.Config, *http.Request) error
}

// NewServer creates a websocket server.
func NewServer(pattern string, cfg tp.PeerConfig, globalLeftPlugin ...tp.Plugin) *Server {
	p := tp.NewPeer(cfg, globalLeftPlugin...)
	serveMux := http.NewServeMux()
	return &Server{
		Peer:     p,
		cfg:      cfg,
		serveMux: serveMux,
		pattern:  pattern,
		server:   &http.Server{Addr: cfg.ListenerAddr(), Handler: serveMux},
	}
}

// ListenAndServeJSON listen and serve with the JSON protocol.
func (srv *Server) ListenAndServeJSON() error {
	return srv.ListenAndServe(jsonSubProto.NewJSONSubProtoFunc())
}

// ListenAndServeProtobuf listen and serve with the Protobuf protocol.
func (srv *Server) ListenAndServeProtobuf() error {
	return srv.ListenAndServe(pbSubProto.NewPbSubProtoFunc())
}

// ListenAndServe listens on the TCP network address addr and then calls
// Serve with handler to handle requests on incoming connections.
// Accepted connections are configured to enable TCP keep-alives.
//
// The handler is typically nil, in which case the DefaultServeMux is used.
//
// ListenAndServe always returns a non-nil error.
//
// If protoFunc is empty, JSON is used by default.
func (srv *Server) ListenAndServe(protoFunc ...tp.ProtoFunc) (err error) {
	network := srv.cfg.Network
	switch network {
	default:
		return errors.New("Invalid network config, refer to the following: tcp, tcp4, tcp6")
	case "tcp", "tcp4", "tcp6":
	}
	srv.Handle(srv.pattern, NewServeHandler(srv.Peer, srv.handshake, protoFunc...))
	addr := srv.cfg.ListenerAddr()
	if addr == "" {
		if srv.Peer.TLSConfig() != nil {
			addr = ":https"
		} else {
			addr = ":http"
		}
	}
	srv.lis, err = tp.NewInheritedListener(network, addr, srv.Peer.TLSConfig())
	if err != nil {
		return
	}
	addr = srv.lis.Addr().String()
	tp.Printf("listen and serve (network:%s, addr:%s)", network, addr)
	for _, v := range srv.Peer.PluginContainer().GetAll() {
		if p, ok := v.(tp.PostListenPlugin); ok {
			p.PostListen(srv.lis.Addr())
		}
	}
	return srv.server.Serve(srv.lis)
}

// Close closes the server.
func (srv *Server) Close() error {
	err := srv.server.Shutdown(context.Background())
	if err != nil {
		srv.Peer.Close()
		return err
	}
	return srv.Peer.Close()
}

// SetHandshake sets customized handshake function.
func (srv *Server) SetHandshake(handshake func(*ws.Config, *http.Request) error) {
	srv.handshake = handshake
}

// Handle registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func (srv *Server) Handle(pattern string, handler http.Handler) {
	srv.serveMux.Handle(pattern, handler)
}

// HandleFunc registers the handler function for the given pattern.
func (srv *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	srv.serveMux.HandleFunc(pattern, handler)
}

// NewJSONServeHandler creates a websocket json handler.
func NewJSONServeHandler(peer tp.Peer, handshake func(*ws.Config, *http.Request) error) http.Handler {
	return NewServeHandler(peer, handshake, jsonSubProto.NewJSONSubProtoFunc())
}

// NewPbServeHandler creates a websocket protobuf handler.
func NewPbServeHandler(peer tp.Peer, handshake func(*ws.Config, *http.Request) error) http.Handler {
	return NewServeHandler(peer, handshake, pbSubProto.NewPbSubProtoFunc())
}

// NewServeHandler creates a websocket handler.
func NewServeHandler(peer tp.Peer, handshake func(*ws.Config, *http.Request) error, protoFunc ...tp.ProtoFunc) http.Handler {
	w := &serverHandler{
		peer:      peer,
		Server:    new(ws.Server),
		protoFunc: NewWsProtoFunc(protoFunc...),
	}
	var scheme string
	if peer.TLSConfig() == nil {
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
		TLSConfig: peer.TLSConfig(),
	}
	return w
}

type serverHandler struct {
	peer      tp.Peer
	protoFunc tp.ProtoFunc
	*ws.Server
}

func (w *serverHandler) handler(conn *ws.Conn) {
	sess, err := w.peer.ServeConn(conn, w.protoFunc)
	if err != nil {
		tp.Errorf("serverHandler: %v", err)
	}
	<-sess.CloseNotify()
}
