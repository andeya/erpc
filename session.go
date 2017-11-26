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

package tp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/coarsetime"
	"github.com/henrylee2cn/goutil/errors"
	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/socket"
)

// Session a connection session.
type (
	PreSession interface {
		// SetId sets the session id.
		SetId(newId string)
		// Close closes the session.
		Close() error
		// Id returns the session id.
		Id() string
		// IsOk checks if the session is ok.
		IsOk() bool
		// Peer returns the peer.
		Peer() *Peer
		// RemoteIp returns the remote peer ip.
		RemoteIp() string
		// LocalIp returns the local peer ip.
		LocalIp() string
		// ReadTimeout returns readdeadline for underlying net.Conn.
		SetReadTimeout(duration time.Duration)
		// WriteTimeout returns writedeadline for underlying net.Conn.
		SetWriteTimeout(duration time.Duration)
		// Public returns temporary public data of session(socket).
		Public() goutil.Map
		// PublicLen returns the length of public data of session(socket).
		PublicLen() int
		// Send sends packet to peer.
		Send(packet *socket.Packet) error
		// Receive receives a packet from peer.
		Receive(packet *socket.Packet) error
	}
	Session interface {
		// SetId sets the session id.
		SetId(newId string)
		// Close closes the session.
		Close() error
		// Id returns the session id.
		Id() string
		// IsOk checks if the session is ok.
		IsOk() bool
		// Peer returns the peer.
		Peer() *Peer
		// GoPull sends a packet and receives reply asynchronously.
		// If the args is []byte or *[]byte type, it can automatically fill in the body codec name.
		GoPull(uri string, args interface{}, reply interface{}, done chan *PullCmd, setting ...socket.PacketSetting)
		// Pull sends a packet and receives reply.
		// If the args is []byte or *[]byte type, it can automatically fill in the body codec name.
		Pull(uri string, args interface{}, reply interface{}, setting ...socket.PacketSetting) *PullCmd
		// Push sends a packet, but do not receives reply.
		// If the args is []byte or *[]byte type, it can automatically fill in the body codec name.
		Push(uri string, args interface{}, setting ...socket.PacketSetting) *Rerror
		// ReadTimeout returns readdeadline for underlying net.Conn.
		ReadTimeout() time.Duration
		// RemoteIp returns the remote peer ip.
		RemoteIp() string
		// LocalIp returns the local peer ip.
		LocalIp() string
		// ReadTimeout returns readdeadline for underlying net.Conn.
		SetReadTimeout(duration time.Duration)
		// WriteTimeout returns writedeadline for underlying net.Conn.
		SetWriteTimeout(duration time.Duration)
		// Socket returns the Socket.
		// Socket() socket.Socket
		// WriteTimeout returns writedeadline for underlying net.Conn.
		WriteTimeout() time.Duration
		// Public returns temporary public data of session(socket).
		Public() goutil.Map
		// PublicLen returns the length of public data of session(socket).
		PublicLen() int
	}
	PostSession interface {
		// Id returns the session id.
		Id() string
		// Peer returns the peer.
		Peer() *Peer
		// RemoteIp returns the remote peer ip.
		RemoteIp() string
		// LocalIp returns the local peer ip.
		LocalIp() string
		// Public returns temporary public data of session(socket).
		Public() goutil.Map
		// PublicLen returns the length of public data of session(socket).
		PublicLen() int
	}
	session struct {
		peer                  *Peer
		pullRouter            *Router
		pushRouter            *Router
		pushSeq               uint64
		pushSeqLock           sync.Mutex
		pullSeq               uint64
		pullCmdMap            goutil.Map
		pullSeqLock           sync.Mutex
		socket                socket.Socket
		closed                int32 // 0:false, 1:true
		disconnected          int32 // 0:false, 1:true
		closeLock             sync.RWMutex
		disconnectLock        sync.RWMutex
		writeLock             sync.Mutex
		graceCtxWaitGroup     sync.WaitGroup
		gracePullCmdWaitGroup sync.WaitGroup
		readTimeout           int32 // time.Duration
		writeTimeout          int32 // time.Duration
	}
)

var (
	_ PreSession  = new(session)
	_ Session     = new(session)
	_ PostSession = new(session)
)

func newSession(peer *Peer, conn net.Conn, protoFuncs []socket.ProtoFunc) *session {
	var s = &session{
		peer:         peer,
		pullRouter:   peer.PullRouter,
		pushRouter:   peer.PushRouter,
		socket:       socket.NewSocket(conn, protoFuncs...),
		pullCmdMap:   goutil.RwMap(),
		readTimeout:  peer.defaultReadTimeout,
		writeTimeout: peer.defaultWriteTimeout,
	}
	return s
}

// Peer returns the peer.
func (s *session) Peer() *Peer {
	return s.peer
}

// Socket returns the Socket.
// func (s *session) Socket() socket.Socket {
// 	return s.socket
// }

// Id returns the session id.
func (s *session) Id() string {
	return s.socket.Id()
}

// SetId sets the session id.
func (s *session) SetId(newId string) {
	oldId := s.Id()
	if oldId == newId {
		return
	}
	s.socket.SetId(newId)
	hub := s.peer.sessHub
	hub.Set(s)
	hub.Delete(oldId)
	Tracef("session changes id: %s -> %s", oldId, newId)
}

// RemoteIp returns the remote peer ip.
func (s *session) RemoteIp() string {
	return s.socket.RemoteAddr().String()
}

// LocalIp returns the local peer ip.
func (s *session) LocalIp() string {
	return s.socket.LocalAddr().String()
}

// ReadTimeout returns readdeadline for underlying net.Conn.
func (s *session) ReadTimeout() time.Duration {
	return time.Duration(atomic.LoadInt32(&s.readTimeout))
}

// WriteTimeout returns writedeadline for underlying net.Conn.
func (s *session) WriteTimeout() time.Duration {
	return time.Duration(atomic.LoadInt32(&s.writeTimeout))
}

// ReadTimeout returns readdeadline for underlying net.Conn.
func (s *session) SetReadTimeout(duration time.Duration) {
	atomic.StoreInt32(&s.readTimeout, int32(duration))
}

// WriteTimeout returns writedeadline for underlying net.Conn.
func (s *session) SetWriteTimeout(duration time.Duration) {
	atomic.StoreInt32(&s.writeTimeout, int32(duration))
}

// Send sends packet to peer.
func (s *session) Send(packet *socket.Packet) error {
	return s.socket.WritePacket(packet)
}

// Receive receives a packet from peer.
func (s *session) Receive(packet *socket.Packet) error {
	return s.socket.ReadPacket(packet)
}

// GoPull sends a packet and receives reply asynchronously.
// If the args is []byte or *[]byte type, it can automatically fill in the body codec name.
func (s *session) GoPull(uri string, args interface{}, reply interface{}, done chan *PullCmd, setting ...socket.PacketSetting) {
	if done == nil && cap(done) == 0 {
		// It must arrange that done has enough buffer for the number of simultaneous
		// RPCs that will be using that channel. If the channel
		// is totally unbuffered, it's best not to run at all.
		Panicf("*session.GoPull(): done channel is unbuffered")
	}
	s.pullSeqLock.Lock()
	seq := s.pullSeq
	s.pullSeq++
	s.pullSeqLock.Unlock()
	output := socket.NewPacket(
		socket.WithSeq(seq),
		socket.WithPtype(TypePull),
		socket.WithUri(uri),
		socket.WithBody(args),
	)
	for _, fn := range setting {
		fn(output)
	}
	if output.BodyCodec() == codec.NilCodecId {
		output.SetBodyCodec(s.peer.defaultBodyCodec)
	}

	cmd := &PullCmd{
		sess:     s,
		output:   output,
		reply:    reply,
		doneChan: done,
		start:    s.peer.timeNow(),
		public:   goutil.RwMap(),
	}

	{
		// count pull-launch
		s.gracePullCmdWaitGroup.Add(1)
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
	s.pullCmdMap.Store(output.Seq(), cmd)
	cmd.rerr = s.peer.pluginContainer.PreWritePull(cmd)
	if cmd.rerr != nil {
		cmd.done()
		return
	}
	if err := s.write(output); err != nil {
		cmd.rerr = rerror_writeFailed.Copy()
		cmd.rerr.Detail = err.Error()
		cmd.done()
		return
	}
	s.peer.pluginContainer.PostWritePull(cmd)
}

// Pull sends a packet and receives reply.
// If the args is []byte or *[]byte type, it can automatically fill in the body codec name.
func (s *session) Pull(uri string, args interface{}, reply interface{}, setting ...socket.PacketSetting) *PullCmd {
	doneChan := make(chan *PullCmd, 1)
	s.GoPull(uri, args, reply, doneChan, setting...)
	pullCmd := <-doneChan
	close(doneChan)
	return pullCmd
}

// Push sends a packet, but do not receives reply.
// If the args is []byte or *[]byte type, it can automatically fill in the body codec name.
func (s *session) Push(uri string, args interface{}, setting ...socket.PacketSetting) *Rerror {
	start := s.peer.timeNow()

	s.pushSeqLock.Lock()
	ctx := s.peer.getContext(s, true)
	output := ctx.output
	output.SetSeq(s.pushSeq)
	s.pushSeq++
	s.pushSeqLock.Unlock()

	ctx.start = start

	output.SetPtype(TypePush)
	output.SetUri(uri)
	output.SetBody(args)

	for _, fn := range setting {
		fn(output)
	}
	if output.BodyCodec() == codec.NilCodecId {
		output.SetBodyCodec(s.peer.defaultBodyCodec)
	}

	defer func() {
		if p := recover(); p != nil {
			Errorf("panic when pushing:\n%v\n%s", p, goutil.PanicTrace(1))
		}
		s.peer.putContext(ctx, true)
	}()
	rerr := s.peer.pluginContainer.PreWritePush(ctx)
	if rerr != nil {
		return rerr
	}
	if err := s.write(output); err != nil {
		rerr = rerror_writeFailed.Copy()
		rerr.Detail = err.Error()
		return rerr
	}
	s.runlog(s.peer.timeSince(ctx.start), nil, output, typePushLaunch)
	s.peer.pluginContainer.PostWritePush(ctx)
	return nil
}

// Public returns temporary public data of session(socket).
func (s *session) Public() goutil.Map {
	return s.socket.Public()
}

// PublicLen returns the length of public data of session(socket).
func (s *session) PublicLen() int {
	return s.socket.PublicLen()
}

func (s *session) startReadAndHandle() {
	var (
		err         error
		readTimeout time.Duration
	)
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%v\n%s", p, goutil.PanicTrace(2))
		}
		s.readDisconnected(err)
		s.Close()
	}()

	// read pull, pull reple or push
	for s.IsOk() {
		var ctx = s.peer.getContext(s, false)
		if s.peer.pluginContainer.PreReadHeader(ctx) != nil {
			s.peer.putContext(ctx, false)
			return
		}

		if readTimeout = s.ReadTimeout(); readTimeout > 0 {
			s.socket.SetReadDeadline(coarsetime.CeilingTimeNow().Add(readTimeout))
		}

		err = s.socket.ReadPacket(ctx.input)
		if err != nil {
			s.peer.putContext(ctx, false)
			// DEBUG:
			// Debugf("s.socket.ReadPacket(ctx.input): err: %v", err)
			return
		}
		if !s.IsOk() {
			// DEBUG:
			// Debugf("s.socket.ReadPacket(ctx.input): s.IsOk()==false")
			s.peer.putContext(ctx, false)
			return
		}

		s.graceCtxWaitGroup.Add(1)
		if !Go(func() {
			defer func() {
				s.peer.putContext(ctx, true)
				if p := recover(); p != nil {
					Debugf("panic:\n%v\n%s", p, goutil.PanicTrace(1))
				}
			}()
			ctx.handle()
		}) {
			s.peer.putContext(ctx, true)
		}
	}
}

// ErrConnClosed connection is closed error.
var ErrConnClosed = errors.New("connection is closed")

func (s *session) write(packet *socket.Packet) (err error) {
	if s.isDisconnected() {
		return ErrConnClosed
	}
	var (
		writeTimeout = s.WriteTimeout()
		now          time.Time
	)
	if writeTimeout > 0 {
		now = coarsetime.CeilingTimeNow()
	}

	s.writeLock.Lock()

	if s.isDisconnected() {
		s.writeLock.Unlock()
		return ErrConnClosed
	}

	defer func() {
		s.writeLock.Unlock()
		if err == io.EOF || err == socket.ErrProactivelyCloseSocket {
			err = ErrConnClosed
		}
	}()

	if writeTimeout > 0 {
		s.socket.SetWriteDeadline(now.Add(writeTimeout))
	}
	err = s.socket.WritePacket(packet)
	return err
}

// IsOk checks if the session is ok.
func (s *session) IsOk() bool {
	return atomic.LoadInt32(&s.disconnected) != 1 && atomic.LoadInt32(&s.closed) != 1
}

// isDisconnected checks if the session is ok.
func (s *session) isDisconnected() bool {
	return atomic.LoadInt32(&s.disconnected) == 1
}

var (
	deadlineForEndlessRead = time.Time{}
	deadlineForStopRead    = time.Time{}.Add(1)
)

// Close closes the session.
func (s *session) Close() (err error) {
	s.closeLock.Lock()
	if atomic.LoadInt32(&s.closed) == 1 {
		s.closeLock.Unlock()
		return nil
	}
	atomic.StoreInt32(&s.closed, 1)

	defer func() {
		s.graceCtxWaitGroup.Wait()
		s.gracePullCmdWaitGroup.Wait()
		// make sure return s.startReadAndHandle
		s.socket.SetReadDeadline(deadlineForStopRead)
		err = s.socket.Close()
		s.peer.sessHub.Delete(s.Id())
		s.closeLock.Unlock()
		s.peer.pluginContainer.PostDisconnect(s)
	}()

	if !s.isDisconnected() {
		// make sure do not disconnect because of reading timeout.
		// wait for the subsequent write to complete.
		s.socket.SetReadDeadline(deadlineForEndlessRead)
	}

	return
}

func (s *session) readDisconnected(err error) {
	if atomic.LoadInt32(&s.closed) == 1 || atomic.LoadInt32(&s.disconnected) == 1 {
		return
	}
	s.disconnectLock.Lock()
	defer s.disconnectLock.Unlock()
	if atomic.LoadInt32(&s.closed) == 1 || atomic.LoadInt32(&s.disconnected) == 1 {
		return
	}

	atomic.StoreInt32(&s.disconnected, 1)

	if err != nil && err != io.EOF && err != socket.ErrProactivelyCloseSocket {
		Debugf("disconnect(%s) when reading: %s", s.RemoteIp(), err.Error())
	}

	s.graceCtxWaitGroup.Wait()

	s.pullCmdMap.Range(func(_, v interface{}) bool {
		pullCmd := v.(*PullCmd)
		pullCmd.cancel()
		return true
	})
}

// SessionHub sessions hub
type SessionHub struct {
	// key: session id (ip, name and so on)
	// value: *session
	sessions goutil.Map
}

// newSessionHub creates a new sessions hub.
func newSessionHub() *SessionHub {
	chub := &SessionHub{
		sessions: goutil.AtomicMap(),
	}
	return chub
}

// Set sets a *session.
func (sh *SessionHub) Set(sess *session) {
	_sess, loaded := sh.sessions.LoadOrStore(sess.Id(), sess)
	if !loaded {
		return
	}
	sh.sessions.Store(sess.Id(), sess)
	if oldSess := _sess.(*session); sess != oldSess {
		oldSess.Close()
	}
}

// Get gets *session by id.
// If second returned arg is false, mean the *session is not found.
func (sh *SessionHub) Get(id string) (*session, bool) {
	_sess, ok := sh.sessions.Load(id)
	if !ok {
		return nil, false
	}
	return _sess.(*session), true
}

// Range calls f sequentially for each id and *session present in the session hub.
// If fn returns false, stop traversing.
func (sh *SessionHub) Range(fn func(*session) bool) {
	sh.sessions.Range(func(key, value interface{}) bool {
		return fn(value.(*session))
	})
}

// Random gets a *session randomly.
// If third returned arg is false, mean no *session is exist.
func (sh *SessionHub) Random() (*session, bool) {
	_, sess, exist := sh.sessions.Random()
	if !exist {
		return nil, false
	}
	return sess.(*session), true
}

// Len returns the length of the session hub.
// Note: the count implemented using sync.Map may be inaccurate.
func (sh *SessionHub) Len() int {
	return sh.sessions.Len()
}

// Delete deletes the *session for a id.
func (sh *SessionHub) Delete(id string) {
	sh.sessions.Delete(id)
}

const (
	typePushLaunch int8 = 1
	typePushHandle int8 = 2
	typePullLaunch int8 = 3
	typePullHandle int8 = 4
)

// func isPushLaunch(input, output *socket.Packet) bool {
// 	return input == nil || (output != nil && output.Ptype() == TypePush)
// }
// func isPushHandle(input, output *socket.Packet) bool {
// 	return output == nil || (input != nil && input.Ptype() == TypePush)
// }
// func isPullLaunch(input, output *socket.Packet) bool {
// 	return output != nil && output.Ptype() == TypePull
// }
// func isPullHandle(input, output *socket.Packet) bool {
// 	return output != nil && output.Ptype() == TypeReply
// }

func (s *session) runlog(costTime time.Duration, input, output *socket.Packet, logType int8) {
	if s.peer.countTime {
		var (
			printFunc func(string, ...interface{})
			slowStr   string
		)
		if costTime < s.peer.slowCometDuration {
			printFunc = Infof
		} else {
			printFunc = Warnf
			slowStr = "(slow)"
		}

		switch logType {
		case typePushLaunch:
			if s.peer.printBody {
				logformat := "[push-launch] remote-ip: %s | seq: %d | cost-time: %s%s | uri: %-30s |\nSEND:\n size: %d\n body[-json]: %s\n"
				printFunc(logformat, s.RemoteIp(), output.Seq(), costTime, slowStr, output.Uri(), output.Size(), bodyLogBytes(output))

			} else {
				logformat := "[push-launch] remote-ip: %s | seq: %d | cost-time: %s%s | uri: %-30s |\nSEND:\n size: %d\n"
				printFunc(logformat, s.RemoteIp(), output.Seq(), costTime, slowStr, output.Uri(), output.Size())
			}

		case typePushHandle:
			if s.peer.printBody {
				logformat := "[push-handle] remote-ip: %s | seq: %d | cost-time: %s%s | uri: %-30s |\nRECV:\n size: %d\n body[-json]: %s\n"
				printFunc(logformat, s.RemoteIp(), input.Seq(), costTime, slowStr, input.Uri(), input.Size(), bodyLogBytes(input))
			} else {
				logformat := "[push-handle] remote-ip: %s | seq: %d | cost-time: %s%s | uri: %-30s |\nRECV:\n size: %d\n"
				printFunc(logformat, s.RemoteIp(), input.Seq(), costTime, slowStr, input.Uri(), input.Size())
			}

		case typePullLaunch:
			if s.peer.printBody {
				logformat := "[pull-launch] remote-ip: %s | seq: %d | cost-time: %s%s | uri: %-30s |\nSEND:\n size: %d\n body[-json]: %s\nRECV:\n size: %d\n status: %s\n body[-json]: %s\n"
				printFunc(logformat, s.RemoteIp(), output.Seq(), costTime, slowStr, output.Uri(), output.Size(), bodyLogBytes(output), input.Size(), getRerrorBytes(input.Meta()), bodyLogBytes(input))
			} else {
				logformat := "[pull-launch] remote-ip: %s | seq: %d | cost-time: %s%s | uri: %-30s |\nSEND:\n size: %d\nRECV:\n size: %d\n status: %s\n"
				printFunc(logformat, s.RemoteIp(), output.Seq(), costTime, slowStr, output.Uri(), output.Size(), input.Size(), getRerrorBytes(input.Meta()))
			}

		case typePullHandle:
			if s.peer.printBody {
				logformat := "[pull-handle] remote-ip: %s | seq: %d | cost-time: %s%s | uri: %-30s |\nRECV:\n size: %d\n body[-json]: %s\nSEND:\n size: %d\n status: %s\n body[-json]: %s\n"
				printFunc(logformat, s.RemoteIp(), input.Seq(), costTime, slowStr, input.Uri(), input.Size(), bodyLogBytes(input), output.Size(), getRerrorBytes(output.Meta()), bodyLogBytes(output))
			} else {
				logformat := "[pull-handle] remote-ip: %s | seq: %d | cost-time: %s%s | uri: %-30s |\nRECV:\n size: %d\nSEND:\n size: %d\n status: %s\n"
				printFunc(logformat, s.RemoteIp(), input.Seq(), costTime, slowStr, input.Uri(), input.Size(), output.Size(), getRerrorBytes(output.Meta()))
			}
		}
	} else {
		switch logType {
		case typePushLaunch:
			if s.peer.printBody {
				logformat := "[push-launch] remote-ip: %s | seq: %d | uri: %-30s |\nSEND:\n size: %d\n body[-json]: %s\n"
				Infof(logformat, s.RemoteIp(), output.Seq(), output.Uri(), output.Size(), bodyLogBytes(output))

			} else {
				logformat := "[push-launch] remote-ip: %s | seq: %d | uri: %-30s |\nSEND:\n size: %d\n"
				Infof(logformat, s.RemoteIp(), output.Seq(), output.Uri(), output.Size())
			}

		case typePushHandle:
			if s.peer.printBody {
				logformat := "[push-handle] remote-ip: %s | seq: %d | uri: %-30s |\nRECV:\n size: %d\n body[-json]: %s\n"
				Infof(logformat, s.RemoteIp(), input.Seq(), input.Uri(), input.Size(), bodyLogBytes(input))
			} else {
				logformat := "[push-handle] remote-ip: %s | seq: %d | uri: %-30s |\nRECV:\n size: %d\n"
				Infof(logformat, s.RemoteIp(), input.Seq(), input.Uri(), input.Size())
			}

		case typePullLaunch:
			if s.peer.printBody {
				logformat := "[pull-launch] remote-ip: %s | seq: %d | uri: %-30s |\nSEND:\n size: %d\n body[-json]: %s\nRECV:\n size: %d\n status: %s\n body[-json]: %s\n"
				Infof(logformat, s.RemoteIp(), output.Seq(), output.Uri(), output.Size(), bodyLogBytes(output), input.Size(), getRerrorBytes(input.Meta()), bodyLogBytes(input))
			} else {
				logformat := "[pull-launch] remote-ip: %s | seq: %d | uri: %-30s |\nSEND:\n size: %d\nRECV:\n size: %d\n status: %s\n"
				Infof(logformat, s.RemoteIp(), output.Seq(), output.Uri(), output.Size(), input.Size(), getRerrorBytes(input.Meta()))
			}

		case typePullHandle:
			if s.peer.printBody {
				logformat := "[pull-handle] remote-ip: %s | seq: %d | uri: %-30s |\nRECV:\n size: %d\n body[-json]: %s\nSEND:\n size: %d\n status: %s\n body[-json]: %s\n"
				Infof(logformat, s.RemoteIp(), input.Seq(), input.Uri(), input.Size(), bodyLogBytes(input), output.Size(), getRerrorBytes(output.Meta()), bodyLogBytes(output))
			} else {
				logformat := "[pull-handle] remote-ip: %s | seq: %d | uri: %-30s |\nRECV:\n size: %d\nSEND:\n size: %d\n status: %s\n"
				Infof(logformat, s.RemoteIp(), input.Seq(), input.Uri(), input.Size(), output.Size(), getRerrorBytes(output.Meta()))
			}
		}
	}
}

func bodyLogBytes(packet *socket.Packet) []byte {
	switch v := packet.Body().(type) {
	case []byte:
		if len(v) == 0 || !isJsonBody(packet) {
			return v
		}
		buf := bytes.NewBuffer(make([]byte, 0, len(v)-1))
		err := json.Indent(buf, v[1:], "", "  ")
		if err != nil {
			return v
		}
		return buf.Bytes()
	case *[]byte:
		if len(*v) == 0 || !isJsonBody(packet) {
			return *v
		}
		buf := bytes.NewBuffer(make([]byte, 0, len(*v)-1))
		err := json.Indent(buf, (*v)[1:], "", "  ")
		if err != nil {
			return *v
		}
		return buf.Bytes()
	default:
		b, _ := json.MarshalIndent(v, " ", "  ")
		return b
	}
}

func isJsonBody(packet *socket.Packet) bool {
	if packet != nil && packet.BodyCodec() == codec.ID_JSON {
		return true
	}
	return false
}
