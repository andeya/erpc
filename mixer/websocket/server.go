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

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/mixer/websocket/jsonSubProto"
	"github.com/henrylee2cn/erpc/v6/mixer/websocket/pbSubProto"
	ws "github.com/henrylee2cn/erpc/v6/mixer/websocket/websocket"
)

// Server a websocket server
type Server struct {
	erpc.Peer
	cfg       erpc.PeerConfig
	serveMux  *http.ServeMux
	server    *http.Server
	rootPath  string
	lis       net.Listener
	lisAddr   net.Addr
	handshake func(*ws.Config, *http.Request) error
}

// NewServer creates a websocket server.
func NewServer(rootPath string, cfg erpc.PeerConfig, globalLeftPlugin ...erpc.Plugin) *Server {
	p := erpc.NewPeer(cfg, globalLeftPlugin...)
	serveMux := http.NewServeMux()
	lisAddr := cfg.ListenAddr()
	host, port, _ := net.SplitHostPort(lisAddr.String())
	if port == "0" {
		if p.TLSConfig() != nil {
			port = "https"
		} else {
			port = "http"
		}
		lisAddr = erpc.NewFakeAddr(lisAddr.Network(), host, port)
	}
	return &Server{
		Peer:     p,
		cfg:      cfg,
		serveMux: serveMux,
		rootPath: fixRootPath(rootPath),
		lisAddr:  lisAddr,
		server:   &http.Server{Addr: lisAddr.String(), Handler: serveMux},
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
func (srv *Server) ListenAndServe(protoFunc ...erpc.ProtoFunc) (err error) {
	network := srv.cfg.Network
	switch network {
	default:
		return errors.New("Invalid network config, refer to the following: tcp, tcp4, tcp6")
	case "tcp", "tcp4", "tcp6":
	}
	srv.Handle(srv.rootPath, NewServeHandler(srv.Peer, srv.handshake, protoFunc...))
	srv.lis, err = erpc.NewInheritedListener(srv.lisAddr, srv.Peer.TLSConfig())
	if err != nil {
		return
	}
	srv.lisAddr = srv.lis.Addr()
	erpc.Printf("listen and serve (network:%s, addr:%s)", network, srv.lisAddr)
	for _, v := range srv.Peer.PluginContainer().GetAll() {
		if p, ok := v.(erpc.PostListenPlugin); ok {
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

// Handle registers the handler for the given rootPath.
// If a handler already exists for rootPath, Handle panics.
func (srv *Server) Handle(rootPath string, handler http.Handler) {
	srv.serveMux.Handle(rootPath, handler)
}

// HandleFunc registers the handler function for the given rootPath.
func (srv *Server) HandleFunc(rootPath string, handler func(http.ResponseWriter, *http.Request)) {
	srv.serveMux.HandleFunc(rootPath, handler)
}

// NewJSONServeHandler creates a websocket json handler.
func NewJSONServeHandler(peer erpc.Peer, handshake func(*ws.Config, *http.Request) error) http.Handler {
	return NewServeHandler(peer, handshake, jsonSubProto.NewJSONSubProtoFunc())
}

// NewPbServeHandler creates a websocket protobuf handler.
func NewPbServeHandler(peer erpc.Peer, handshake func(*ws.Config, *http.Request) error) http.Handler {
	return NewServeHandler(peer, handshake, pbSubProto.NewPbSubProtoFunc())
}

// NewServeHandler creates a websocket handler.
func NewServeHandler(peer erpc.Peer, handshake func(*ws.Config, *http.Request) error, protoFunc ...erpc.ProtoFunc) http.Handler {
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
	peer      erpc.Peer
	protoFunc erpc.ProtoFunc
	*ws.Server
}

func (w *serverHandler) handler(conn *ws.Conn) {
	sess, err := w.peer.ServeConn(conn, w.protoFunc)
	if err != nil {
		erpc.Errorf("serverHandler: %v", err)
		return
	}
	<-sess.CloseNotify()
}
