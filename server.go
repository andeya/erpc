// Copyright 2015-2017 HenryLee. All Rights Reserved.
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

package teleport

import (
	"crypto/tls"
	"net"
	"sync"
	"time"

	"github.com/henrylee2cn/goutil/errors"
)

type (
	Server struct {
		peer      *Peer
		addr      string
		tlsConfig *tls.Config
		lis       net.Listener
		mu        sync.Mutex
	}
	SrvConfig struct {
		Id                   string
		Addr                 string
		ReadTimeout          time.Duration // readdeadline for underlying net.Conn
		WriteTimeout         time.Duration // writedeadline for underlying net.Conn
		TlsCertFile          string
		TlsKeyFile           string
		SlowResponseDuration time.Duration // ns,µs,ms,s ...
	}
)

// ErrServerClosed server closed error.
var ErrServerClosed = errors.New("teleport: Server closed")

// NewServer creates a new server.
func NewServer(cfg *SrvConfig) *Server {
	tlsConfig, err := newTLSConfig(cfg.TlsCertFile, cfg.TlsKeyFile)
	if err != nil {
		Fatalf("%v", err)
	}
	s := &Server{
		peer:      NewPeer(cfg.Id, cfg.ReadTimeout, cfg.WriteTimeout),
		addr:      cfg.Addr,
		tlsConfig: tlsConfig,
	}
	return s
}

// Group add api group.
func (s *Server) Group(pathPrefix string, plugins ...Plugin) *ApiMap {
	return s.peer.ApiMap.Group(pathPrefix, plugins...)
}

// Reg registers api.
func (s *Server) Reg(pathPrefix string, ctrlStruct Context, plugin ...Plugin) {
	err := s.peer.ApiMap.Reg(pathPrefix, ctrlStruct, plugin...)
	if err != nil {
		Fatalf("%v", err)
	}
}

func (s *Server) Server() error {
	var err error
	if s.tlsConfig != nil {
		s.lis, err = tls.Listen("tcp", s.addr, s.tlsConfig)
	} else {
		s.lis, err = net.Listen("tcp", s.addr)
	}
	if err != nil {
		Fatalf("%v", err)
	}
	defer s.lis.Close()

	var (
		tempDelay time.Duration // how long to sleep on accept failure
		closeCh   = s.peer.closeCh
	)

	for {
		rw, e := s.lis.Accept()
		if e != nil {
			select {
			case <-closeCh:
				return ErrServerClosed
			default:
			}
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				Tracef("teleport: Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0
		s.peer.ServeConn(rw)
	}
}

func (s *Server) Close() {
	s.peer.Close()
}
