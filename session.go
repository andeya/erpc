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

	"github.com/henrylee2cn/goutil/coarsetime"

	"github.com/henrylee2cn/teleport/socket"
)

type Session struct {
	peer   *Peer
	apiMap *ApiMap
	socket socket.Socket
}

func newSession(peer *Peer, conn net.Conn, id ...string) *Session {
	var s = &Session{
		peer:   peer,
		apiMap: peer.ApiMap,
		socket: socket.NewSocket(conn, id...),
	}
	go s.serve()
	return s
}

func (s *Session) serve() {
	var ctx = s.peer.getContext()
	defer func() {
		recover()
		s.peer.putContext(ctx)
		s.Close()
	}()
	var (
		err          error
		readTimeout  = s.peer.readTimeout
		writeTimeout = s.peer.writeTimeout
	)
	for {
		// read request, response or push
		if readTimeout > 0 {
			s.socket.SetReadDeadline(coarsetime.CoarseTimeNow().Add(readTimeout))
		}
		err = s.socket.ReadPacket(ctx.input)
		if err != nil {
			return
		} else {
		}

		// write response
		if writeTimeout > 0 {
			s.socket.SetWriteDeadline(coarsetime.CoarseTimeNow().Add(writeTimeout))
		}
		err = s.socket.WritePacket(ctx.output)
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
