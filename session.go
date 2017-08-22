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
	"io"
	"net"
	"sync"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/coarsetime"
	"github.com/henrylee2cn/goutil/pool"

	"github.com/henrylee2cn/teleport/socket"
)

// Session a connection session.
type Session struct {
	peer          *Peer
	requestRouter *Router
	pushRouter    *Router
	pushSeq       uint64
	pullSeq       uint64
	pullCmdMap    goutil.Map
	socket        socket.Socket
	closed        bool
	writeLock     sync.Mutex
	closedLock    sync.RWMutex
	gopool        *pool.GoPool
}

const (
	maxGoroutinesAmount      = 1024
	maxGoroutineIdleDuration = 10 * time.Second
)

func newSession(peer *Peer, conn net.Conn, id ...string) *Session {
	var s = &Session{
		peer:          peer,
		requestRouter: peer.RequestRouter,
		pushRouter:    peer.PushRouter,
		socket:        socket.NewSocket(conn, id...),
		pullCmdMap:    goutil.RwMap(),
		gopool:        peer.gopool,
	}
	err := s.gopool.Go(s.readAndHandle)
	if err != nil {
		Warnf("%s", err.Error())
	}
	return s
}

// Id returns the session id.
func (s *Session) Id() string {
	return s.socket.Id()
}

// ChangeId changes the session id.
func (s *Session) ChangeId(newId string) {
	oldId := s.Id()
	s.socket.ChangeId(newId)
	s.peer.sessionHub.Set(s)
	s.peer.sessionHub.Delete(oldId)
}

// Ip returns the remote peer ip.
func (s *Session) Ip() string {
	return s.socket.RemoteAddr().String()
}

// PullCmd the command of the pulling operation's response.
type PullCmd struct {
	packet   *socket.Packet
	reply    interface{}
	doneChan chan *PullCmd // Strobes when pull is complete.
	Xerror   Xerror
}

func (p *PullCmd) done() {
	p.doneChan <- p
}

// GoPull sends a packet and receives reply asynchronously.
func (s *Session) GoPull(uri string, args interface{}, reply interface{}, done chan *PullCmd, setting ...socket.PacketSetting) {
	if done == nil && cap(done) == 0 {
		// It must arrange that done has enough buffer for the number of simultaneous
		// RPCs that will be using that channel. If the channel
		// is totally unbuffered, it's best not to run at all.
		Panicf("teleport: *Session.GoPull(): done channel is unbuffered")
	}
	packet := &socket.Packet{
		Header: &socket.Header{
			Seq:  s.pullSeq,
			Uri:  uri,
			Gzip: s.peer.defaultGzipLevel,
		},
		Body:        args,
		HeaderCodec: s.peer.defaultCodec,
		BodyCodec:   s.peer.defaultCodec,
	}
	s.pullSeq++
	for _, f := range setting {
		f(packet)
	}
	cmd := &PullCmd{
		packet:   packet,
		reply:    reply,
		doneChan: done,
	}
	err := s.write(packet)
	if err == nil {
		s.pullCmdMap.Store(packet.Header.Seq, cmd)
	} else {
		cmd.Xerror = NewXerror(StatusWriteFailed, err.Error())
		cmd.done()
	}
}

// Pull sends a packet and receives reply.
func (s *Session) Pull(uri string, args interface{}, reply interface{}, setting ...socket.PacketSetting) *PullCmd {
	doneChan := make(chan *PullCmd, 1)
	s.GoPull(uri, args, reply, doneChan, setting...)
	pullCmd := <-doneChan
	defer func() {
		recover()
	}()
	close(doneChan)
	return pullCmd
}

// Push sends a packet, but do not receives reply.
func (s *Session) Push(uri string, args interface{}) error {
	packet := &socket.Packet{
		Header: &socket.Header{
			Seq:  s.pushSeq,
			Uri:  uri,
			Gzip: s.peer.defaultGzipLevel,
		},
		Body:        args,
		HeaderCodec: s.peer.defaultCodec,
		BodyCodec:   s.peer.defaultCodec,
	}
	s.pushSeq++
	return s.write(packet)
}

// Closed checks if the session is closed.
func (s *Session) Closed() bool {
	s.closedLock.RLock()
	defer s.closedLock.RUnlock()
	return s.closed
}

// Close closes the session.
func (s *Session) Close() error {
	s.closedLock.Lock()
	defer s.closedLock.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	s.pullCmdMap.Range(func(_, v interface{}) bool {
		pullCmd := v.(*PullCmd)
		pullCmd.Xerror = NewXerror(StatusConnClosed, StatusText(StatusConnClosed))
		pullCmd.done()
		return true
	})
	s.pullCmdMap = nil
	return s.socket.Close()
}

func (s *Session) readAndHandle() {
	defer func() {
		if p := recover(); p != nil {
			Debugf("*Session.readAndHandle() panic:\n%v\n%s", p, goutil.PanicTrace(1))
		}
		s.Close()
	}()
	var (
		err         error
		readTimeout = s.peer.readTimeout
	)
	for !s.Closed() {
		var ctx = s.peer.getContext(s)
		// read request, response or push
		if readTimeout > 0 {
			s.socket.SetReadDeadline(coarsetime.CoarseTimeNow().Add(readTimeout))
		}
		err = s.socket.ReadPacket(ctx.input)
		if err != nil {
			s.peer.putContext(ctx)
			if err != io.EOF {
				Debugf("teleport: ReadPacket failed: %s", err.Error())
			}
			return
		}

		err = s.gopool.Go(func() {
			defer s.peer.putContext(ctx)
			switch ctx.input.Header.Type {
			case TypeResponse:
				// handle response
				ctx.respHandle()

			case TypePush:
				// handle push
				Debugf("ctx.handle()")
				ctx.handle()

			case TypeRequest:
				// handle response
				ctx.handle()
				ctx.output.Header.Type = TypeResponse
			}
		})
		if err != nil {
			Warnf("%s", err.Error())
		}
	}
}

func (s *Session) write(packet *socket.Packet) error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	// if s.Closed() {
	// 	return
	// }
	var writeTimeout = s.peer.writeTimeout
	if writeTimeout > 0 {
		s.socket.SetWriteDeadline(coarsetime.CoarseTimeNow().Add(writeTimeout))
	}
	err := s.socket.WritePacket(packet)
	if err != nil {
		s.Close()
	}
	return err
}
