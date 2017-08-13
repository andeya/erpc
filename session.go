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
	"net"
	"reflect"
	"sync"

	"github.com/henrylee2cn/teleport/socket"
)

type Session struct {
	peer        *Peer
	router      *Router
	socket      socket.Socket
	ctxLock     sync.Mutex
	freeContext *context
}

func (s *Session) getContext() *context {
	s.ctxLock.Lock()
	ctx := s.freeContext
	if ctx == nil {
		ctx = newCtx(s)
	} else {
		s.freeContext = ctx.next
		*ctx = context{}
	}
	s.ctxLock.Unlock()
	return ctx
}

func (s *Session) freeContext(ctx *context) {
	s.ctxLock.Lock()
	ctx.next = s.freeContext
	s.freeContext = ctx
	s.ctxLock.Unlock()
}

func NewSession(peer *Peer, conn net.Conn, id ...string) *Session {
	var s = &Session{
		peer:   peer,
		router: peer.router,
		socket: socket.NewSocket(conn, id...),
	}
	go s.serve()
	return s
}

func (s *Session) serve() {
	defer func() {
		recover()
		s.Close()
	}()
	for {
		var ctx = s.getContext()
		// read request, response or push
		n, err := s.socket.ReadPacket(ctx.loadBody)
		if err != nil {
			return
		} else {
		}

		// write response
		_, err = s.WritePacket(header, now)
		if err != nil {
		}
	}
}

func (s *Session) Id() string {
	return s.socket.Id()
}

func (s *Session) Close() error {
	return s.socket.Close()
}

func (s *Session) push()     {}
func (s *Session) request()  {}
func (s *Session) response() {}
