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

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/coarsetime"

	"github.com/henrylee2cn/teleport/socket"
)

// Session a connection session.
type Session struct {
	peer       *Peer
	seq        uint64
	apiMap     *ApiMap
	pullCmdMap goutil.Map
	socket     socket.Socket
}

func newSession(peer *Peer, conn net.Conn, id ...string) *Session {
	var s = &Session{
		peer:       peer,
		apiMap:     peer.ApiMap,
		socket:     socket.NewSocket(conn, id...),
		pullCmdMap: goutil.RwMap(),
	}
	go s.serve()
	return s
}

func (s *Session) Id() string {
	return s.socket.Id()
}

// TODO
func (s *Session) GoPull(uri string, args interface{}, reply interface{}, done chan *PullCmd, packetSetting ...socket.PacketSetting) {
	packet := &socket.Packet{
		Header: &socket.Header{
			Seq:  s.seq,
			Uri:  uri,
			Gzip: s.peer.defaultGzipLevel,
		},
		Body: args,
	}
	s.seq++
	for _, f := range packetSetting {
		f(packet)
	}
	cmd := &PullCmd{
		packet:   packet,
		reply:    reply,
		doneChan: done,
	}
	_ = cmd
}

// Pull requests a service api and receives reply.
func (s *Session) Pull(uri string, args interface{}, reply interface{}) *PullCmd {
	doneChan := make(chan *PullCmd, 1)
	s.GoPull(uri, args, reply, doneChan)
	pullCmd := <-doneChan
	close(doneChan)
	return pullCmd
}

// TODO
func (s *Session) Push() error {
	return nil
}

func (s *Session) Close() error {
	return s.socket.Close()
}

// TODO
func (s *Session) serve() {
	var ctx = s.peer.getContext(s)
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
			Debugf("teleport: ReadPacket: %s", err.Error())
			return
		}
		switch ctx.input.Header.Type {
		case TypeRequest:
			// handle
			go ctx.autoHandle()
			ctx.output.Header.Type = TypeResponse
			// write response
			if writeTimeout > 0 {
				s.socket.SetWriteDeadline(coarsetime.CoarseTimeNow().Add(writeTimeout))
			}
			err = s.socket.WritePacket(ctx.output)
			if err != nil {
				Debugf("teleport: WritePacket: %s", err.Error())
				return
			}
		}
	}
}

func (s *Session) read() {

}

func (s *Session) write() {

}
