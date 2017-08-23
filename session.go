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
	"encoding/json"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/henrylee2cn/go-logging/color"
	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/coarsetime"
	"github.com/henrylee2cn/goutil/errors"
	"github.com/henrylee2cn/goutil/pool"

	"github.com/henrylee2cn/teleport/socket"
)

// Session a connection session.
type Session struct {
	peer           *Peer
	pullRouter     *Router
	pushRouter     *Router
	pushSeq        uint64
	pullSeq        uint64
	pullCmdMap     goutil.Map
	socket         socket.Socket
	closed         int32 // 0:false, 1:true
	disconnected   int32 // 0:false, 1:true
	closeLock      sync.RWMutex
	disconnectLock sync.RWMutex
	writeLock      sync.Mutex
	gopool         *pool.GoPool
	graceWaitGroup sync.WaitGroup
}

const (
	maxGoroutinesAmount      = 1024
	maxGoroutineIdleDuration = 10 * time.Second
)

func newSession(peer *Peer, conn net.Conn, id ...string) *Session {
	var s = &Session{
		peer:       peer,
		pullRouter: peer.PullRouter,
		pushRouter: peer.PushRouter,
		socket:     socket.NewSocket(conn, id...),
		pullCmdMap: goutil.RwMap(),
		gopool:     peer.gopool,
	}
	err := s.gopool.Go(s.readAndHandle)
	if err != nil {
		Warnf("%s", err.Error())
	}
	return s
}

// Peer returns the peer.
func (s *Session) Peer() *Peer {
	return s.peer
}

// Socket returns the Socket.
func (s *Session) Socket() socket.Socket {
	return s.socket
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
	Tracef("session changes id: %s -> %s", oldId, newId)
}

// RemoteIp returns the remote peer ip.
func (s *Session) RemoteIp() string {
	return s.socket.RemoteAddr().String()
}

// GoPull sends a packet and receives reply asynchronously.
func (s *Session) GoPull(uri string, args interface{}, reply interface{}, done chan *PullCmd, setting ...socket.PacketSetting) {
	if done == nil && cap(done) == 0 {
		// It must arrange that done has enough buffer for the number of simultaneous
		// RPCs that will be using that channel. If the channel
		// is totally unbuffered, it's best not to run at all.
		Panicf("*Session.GoPull(): done channel is unbuffered")
	}
	output := &socket.Packet{
		Header: &socket.Header{
			Seq:  s.pullSeq,
			Type: TypePull,
			Uri:  uri,
			Gzip: s.peer.defaultGzipLevel,
		},
		Body:        args,
		HeaderCodec: s.peer.defaultCodec,
		BodyCodec:   s.peer.defaultCodec,
	}
	s.pullSeq++
	for _, f := range setting {
		f(output)
	}

	cmd := &PullCmd{
		session:  s,
		output:   output,
		reply:    reply,
		doneChan: done,
		start:    time.Now(),
		public:   goutil.RwMap(),
	}

	{
		// count pull-launch
		s.graceWaitGroup.Add(1)
	}

	if s.socket.PublicLen() > 0 {
		s.socket.Public().Range(func(key, value interface{}) bool {
			cmd.public.Store(key, value)
			return true
		})
	}

	defer func() {
		if p := recover(); p != nil {
			Errorf("panic:\n%v\n%s", p, goutil.PanicTrace(1))
		}
	}()

	err := s.peer.pluginContainer.PreWritePull(cmd)
	if err == nil {
		if err = s.write(output); err == nil {
			s.pullCmdMap.Store(output.Header.Seq, cmd)
			if err = s.peer.pluginContainer.PostWritePull(cmd); err != nil {
				Errorf("%s", err.Error())
			}
			return
		}
	}
	cmd.Xerror = NewXerror(StatusWriteFailed, err.Error())
	cmd.done()
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
func (s *Session) Push(uri string, args interface{}) (err error) {
	ctx := s.peer.getContext(s)
	ctx.start = time.Now()

	output := ctx.output
	header := output.Header
	header.Seq = s.pushSeq
	header.Type = TypePush
	header.Uri = uri
	header.Gzip = s.peer.defaultGzipLevel

	output.Body = args
	output.HeaderCodec = s.peer.defaultCodec
	output.BodyCodec = s.peer.defaultCodec

	s.pushSeq++

	defer func() {
		if p := recover(); p != nil {
			err = errors.Errorf("panic:\n%v\n%s", p, goutil.PanicTrace(1))
		}
		s.runlog(time.Since(ctx.start), nil, output)
		s.peer.putContext(ctx)
	}()

	err = s.peer.pluginContainer.PreWritePush(ctx)
	if err == nil {
		if err = s.write(output); err == nil {
			err = s.peer.pluginContainer.PostWritePush(ctx)
		}
	}
	return err
}

func (s *Session) readAndHandle() {
	defer func() {
		if p := recover(); p != nil {
			Debugf("panic:\n%v\n%s", p, goutil.PanicTrace(1))
		}
		s.Close()
	}()

	var (
		err         error
		readTimeout = s.peer.readTimeout
	)
	for s.IsOk() {
		var ctx = s.peer.getContext(s)
		// read request, response or push
		if readTimeout > 0 {
			s.socket.SetReadDeadline(coarsetime.CoarseTimeNow().Add(readTimeout))
		}

		err = s.peer.pluginContainer.PreReadHeader(ctx)
		if err != nil {
			s.peer.putContext(ctx)
			Errorf("%s", err.Error())
			s.markDisconnected()
			return
		}

		err = s.socket.ReadPacket(ctx.input)
		if err != nil {
			s.peer.putContext(ctx)
			if err != io.EOF {
				Debugf("ReadPacket() failed: %s", err.Error())
			}
			s.markDisconnected()
			return
		}

		err = s.gopool.Go(func() {
			defer s.peer.putContext(ctx)
			switch ctx.input.Header.Type {
			case TypeReply:
				// handles pull reply
				ctx.handleReply()

			case TypePush:
				//  handles push
				ctx.handle()

			case TypePull:
				// handles and replies pull
				ctx.handle()
				ctx.output.Header.Type = TypeReply
			}
		})
		if err != nil {
			Warnf("%s", err.Error())
		}
	}
}

// ErrConnClosed connection is closed error.
var ErrConnClosed = errors.New("connection is closed")

func (s *Session) write(packet *socket.Packet) (err error) {
	s.writeLock.Lock()
	defer func() {
		if p := recover(); p != nil {
			err = errors.Errorf("panic:\n%v\n%s", p, goutil.PanicTrace(1))
		} else if err == io.EOF {
			err = ErrConnClosed
		}
		s.writeLock.Unlock()
	}()
	// if s.Closed() {
	// 	return
	// }
	var writeTimeout = s.peer.writeTimeout
	if writeTimeout > 0 {
		s.socket.SetWriteDeadline(coarsetime.CoarseTimeNow().Add(writeTimeout))
	}
	err = s.socket.WritePacket(packet)
	if err != nil {
		s.markDisconnected()
	}
	return err
}

// IsOk checks if the session is ok.
func (s *Session) IsOk() bool {
	if atomic.LoadInt32(&s.disconnected) == 1 {
		return false
	}
	if atomic.LoadInt32(&s.closed) == 1 {
		return false
	}
	return true
}

// Close closes the session.
func (s *Session) Close() error {
	if atomic.LoadInt32(&s.closed) == 1 {
		return nil
	}
	atomic.StoreInt32(&s.closed, 1)

	s.closeLock.Lock()
	defer s.closeLock.Unlock()
	s.graceWaitGroup.Wait()
	s.peer.sessionHub.Delete(s.Id())
	s.pullCmdMap = nil
	return s.socket.Close()
}

func (s *Session) markDisconnected() {
	if atomic.LoadInt32(&s.disconnected) == 1 {
		return
	}
	atomic.StoreInt32(&s.disconnected, 1)

	s.disconnectLock.Lock()
	defer s.disconnectLock.Unlock()
	s.pullCmdMap.Range(func(_, v interface{}) bool {
		pullCmd := v.(*PullCmd)
		pullCmd.Xerror = NewXerror(StatusConnClosed, StatusText(StatusConnClosed))
		pullCmd.done()
		return true
	})
}

func isPushLaunch(input, output *socket.Packet) bool {
	return input == nil || output.Header.Type == TypePush
}
func isPushHandle(input, output *socket.Packet) bool {
	return output == nil || input.Header.Type == TypePush
}
func isPullLaunch(input, output *socket.Packet) bool {
	return output.Header.Type == TypePull
}
func isPullHandle(input, output *socket.Packet) bool {
	return output.Header.Type == TypeReply
}

func (s *Session) runlog(costTime time.Duration, input, output *socket.Packet) {
	var (
		printFunc func(string, ...interface{})
		slowStr   string
		logformat string
		printBody = s.peer.printBody
	)
	if costTime < s.peer.slowCometDuration {
		printFunc = Infof
	} else {
		printFunc = Warnf
		slowStr = "(slow)"
	}

	if isPushLaunch(input, output) {
		if printBody {
			logformat = "[push-launch] remote-ip: %s | cost-time: %s%s | uri: %-30s |\nSEND:\n packet-length: %d\n body-json: %s\n"
			printFunc(logformat, s.RemoteIp(), costTime, slowStr, output.Header.Uri, output.Length, bodyLogBytes(output.Body))

		} else {
			logformat = "[push-launch] remote-ip: %s | cost-time: %s%s | uri: %-30s |\nSEND:\n packet-length: %d\n"
			printFunc(logformat, s.RemoteIp(), costTime, slowStr, output.Header.Uri, output.Length)
		}

	} else if isPushHandle(input, output) {
		if printBody {
			logformat = "[push-handle] remote-ip: %s | cost-time: %s%s | uri: %-30s |\nRECV:\n packet-length: %d\n body-json: %s\n"
			printFunc(logformat, s.RemoteIp(), costTime, slowStr, input.Header.Uri, input.Length, bodyLogBytes(input.Body))
		} else {
			logformat = "[push-handle] remote-ip: %s | cost-time: %s%s | uri: %-30s |\nRECV:\n packet-length: %d\n"
			printFunc(logformat, s.RemoteIp(), costTime, slowStr, input.Header.Uri, input.Length)
		}

	} else if isPullLaunch(input, output) {
		if printBody {
			logformat = "[pull-launch] remote-ip: %s | cost-time: %s%s | uri: %-30s |\nSEND:\n packet-length: %d\n body-json: %s\nRECV:\n status: %s %s\n packet-length: %d\n body-json: %s\n"
			printFunc(logformat, s.RemoteIp(), costTime, slowStr, output.Header.Uri, output.Length, bodyLogBytes(output.Body), colorCode(input.Header.StatusCode), input.Header.Status, input.Length, bodyLogBytes(input.Body))
		} else {
			logformat = "[pull-launch] remote-ip: %s | cost-time: %s%s | uri: %-30s |\nSEND:\n packet-length: %d\nRECV:\n status: %s %s\n packet-length: %d\n"
			printFunc(logformat, s.RemoteIp(), costTime, slowStr, output.Header.Uri, output.Length, colorCode(input.Header.StatusCode), input.Header.Status, input.Length)
		}

	} else if isPullHandle(input, output) {
		if printBody {
			logformat = "[pull-handle] remote-ip: %s | cost-time: %s%s | uri: %-30s |\nRECV:\n packet-length: %d\n body-json: %s\nSEND:\n status: %s %s\n packet-length: %d\n body-json: %s\n"
			printFunc(logformat, s.RemoteIp(), costTime, slowStr, input.Header.Uri, input.Length, bodyLogBytes(input.Body), colorCode(output.Header.StatusCode), output.Header.Status, output.Length, bodyLogBytes(output.Body))
		} else {
			logformat = "[pull-handle] remote-ip: %s | cost-time: %s%s | uri: %-30s |\nRECV:\n packet-length: %d\nSEND:\n status: %s %s\n packet-length: %d\n"
			printFunc(logformat, s.RemoteIp(), costTime, slowStr, input.Header.Uri, input.Length, colorCode(output.Header.StatusCode), output.Header.Status, output.Length)
		}
	}
}

func bodyLogBytes(v interface{}) []byte {
	b, _ := json.MarshalIndent(v, " ", "  ")
	return b
}

func colorCode(code int32) string {
	switch {
	case code >= 500 || code < 200:
		return color.Red(code)
	case code >= 400:
		return color.Magenta(code)
	case code >= 300:
		return color.Grey(code)
	default:
		return color.Green(code)
	}
}

// SessionHub sessions hub
type SessionHub struct {
	// key: session id (ip, name and so on)
	// value: *Session
	sessions goutil.Map
}

// newSessionHub creates a new sessions hub.
func newSessionHub() *SessionHub {
	chub := &SessionHub{
		sessions: goutil.AtomicMap(),
	}
	return chub
}

// Set sets a *Session.
func (sh *SessionHub) Set(session *Session) {
	_session, loaded := sh.sessions.LoadOrStore(session.Id(), session)
	if !loaded {
		return
	}
	if oldSession := _session.(*Session); session != oldSession {
		oldSession.Close()
	}
}

// Get gets *Session by id.
// If second returned arg is false, mean the *Session is not found.
func (sh *SessionHub) Get(id string) (*Session, bool) {
	_session, ok := sh.sessions.Load(id)
	if !ok {
		return nil, false
	}
	return _session.(*Session), true
}

// Range calls f sequentially for each id and *Session present in the session hub.
// If f returns false, range stops the iteration.
func (sh *SessionHub) Range(f func(*Session) bool) {
	sh.sessions.Range(func(key, value interface{}) bool {
		return f(value.(*Session))
	})
}

// Random gets a *Session randomly.
// If third returned arg is false, mean no *Session is exist.
func (sh *SessionHub) Random() (*Session, bool) {
	_, session, exist := sh.sessions.Random()
	if !exist {
		return nil, false
	}
	return session.(*Session), true
}

// Len returns the length of the session hub.
// Note: the count implemented using sync.Map may be inaccurate.
func (sh *SessionHub) Len() int {
	return sh.sessions.Len()
}

// Delete deletes the *Session for a id.
func (sh *SessionHub) Delete(id string) {
	sh.sessions.Delete(id)
}
