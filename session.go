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
		PostSession
		// SetId sets the session id.
		SetId(newId string)
		// Send sends packet to peer, before the formal connection.
		// Note:
		// the external setting seq is invalid, the internal will be forced to set;
		// does not support automatic redial after disconnection.
		Send(output *socket.Packet) *Rerror
		// Receive receives a packet from peer, before the formal connection.
		// Note: does not support automatic redial after disconnection.
		Receive(input *socket.Packet) *Rerror
		// ReadTimeout returns readdeadline for underlying net.Conn.
		ReadTimeout() time.Duration
		// ReadTimeout returns readdeadline for underlying net.Conn.
		SetReadTimeout(duration time.Duration)
		// WriteTimeout returns writedeadline for underlying net.Conn.
		SetWriteTimeout(duration time.Duration)
		// WriteTimeout returns writedeadline for underlying net.Conn.
		WriteTimeout() time.Duration
	}
	Session interface {
		PostSession
		// SetId sets the session id.
		SetId(newId string)
		// Close closes the session.
		Close() error
		// Health checks if the session is ok.
		Health() bool
		// AsyncPull sends a packet and receives reply asynchronously.
		// If the args is []byte or *[]byte type, it can automatically fill in the body codec name.
		AsyncPull(uri string, args interface{}, reply interface{}, done chan *PullCmd, setting ...socket.PacketSetting)
		// Pull sends a packet and receives reply.
		// Note:
		// If the args is []byte or *[]byte type, it can automatically fill in the body codec name;
		// If the session is a client role and PeerConfig.RedialTimes>0, it is automatically re-called once after a failure.
		Pull(uri string, args interface{}, reply interface{}, setting ...socket.PacketSetting) *PullCmd
		// Push sends a packet, but do not receives reply.
		// Note:
		// If the args is []byte or *[]byte type, it can automatically fill in the body codec name;
		// If the session is a client role and PeerConfig.RedialTimes>0, it is automatically re-called once after a failure.
		Push(uri string, args interface{}, setting ...socket.PacketSetting) *Rerror
		// ReadTimeout returns readdeadline for underlying net.Conn.
		ReadTimeout() time.Duration
		// ReadTimeout returns readdeadline for underlying net.Conn.
		SetReadTimeout(duration time.Duration)
		// WriteTimeout returns writedeadline for underlying net.Conn.
		SetWriteTimeout(duration time.Duration)
		// WriteTimeout returns writedeadline for underlying net.Conn.
		WriteTimeout() time.Duration
	}
	PostSession interface {
		// Id returns the session id.
		Id() string
		// Peer returns the peer.
		Peer() *Peer
		// LocalIp returns the local peer ip.
		LocalIp() string
		// RemoteIp returns the remote peer ip.
		RemoteIp() string
		// Public returns temporary public data of session(socket).
		Public() goutil.Map
		// PublicLen returns the length of public data of session(socket).
		PublicLen() int
	}

	session struct {
		peer                  *Peer
		pullRouter            *Router
		pushRouter            *Router
		seq                   uint64
		seqLock               sync.Mutex
		pullCmdMap            goutil.Map
		socket                socket.Socket
		status                int32 // 0:ok, 1:active closed, 2:disconnect
		closeLock             sync.RWMutex
		writeLock             sync.Mutex
		graceCtxWaitGroup     sync.WaitGroup
		gracePullCmdWaitGroup sync.WaitGroup
		readTimeout           int32 // time.Duration
		writeTimeout          int32 // time.Duration
		// only for client role
		redialForClientFunc func() bool
		redialLock          sync.Mutex
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
		pullCmdMap:   goutil.AtomicMap(),
		readTimeout:  int32(peer.defaultReadTimeout),
		writeTimeout: int32(peer.defaultWriteTimeout),
	}
	return s
}

// Peer returns the peer.
func (s *session) Peer() *Peer {
	return s.peer
}

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

// Send sends packet to peer, before the formal connection.
// Note:
// the external setting seq is invalid, the internal will be forced to set;
// does not support automatic redial after disconnection.
func (s *session) Send(output *socket.Packet) *Rerror {
	s.seqLock.Lock()
	output.SetSeq(s.seq)
	s.seq++
	s.seqLock.Unlock()
	err := s.socket.WritePacket(output)
	if err != nil {
		rerr := rerror_connClosed.Copy()
		rerr.Detail = err.Error()
		return rerr
	}
	return nil
}

// Receive receives a packet from peer, before the formal connection.
// Note: does not support automatic redial after disconnection.
func (s *session) Receive(input *socket.Packet) *Rerror {
	if readTimeout := s.ReadTimeout(); readTimeout > 0 {
		s.socket.SetReadDeadline(coarsetime.CeilingTimeNow().Add(readTimeout))
	}
	if err := s.socket.ReadPacket(input); err != nil {
		rerr := rerror_connClosed.Copy()
		rerr.Detail = err.Error()
		return rerr
	}
	return nil
}

// AsyncPull sends a packet and receives reply asynchronously.
// Note:
// If the args is []byte or *[]byte type, it can automatically fill in the body codec name;
// If the session is a client role and PeerConfig.RedialTimes>0, it is automatically re-called once after a failure.
func (s *session) AsyncPull(uri string, args interface{}, reply interface{}, done chan *PullCmd, setting ...socket.PacketSetting) {
	if done == nil && cap(done) == 0 {
		// It must arrange that done has enough buffer for the number of simultaneous
		// RPCs that will be using that channel. If the channel
		// is totally unbuffered, it's best not to run at all.
		Panicf("*session.AsyncPull(): done channel is unbuffered")
	}
	s.seqLock.Lock()
	seq := s.seq
	s.seq++
	s.seqLock.Unlock()
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

	// count pull-launch
	s.gracePullCmdWaitGroup.Add(1)

	if s.socket.PublicLen() > 0 {
		s.socket.Public().Range(func(key, value interface{}) bool {
			cmd.public.Store(key, value)
			return true
		})
	}

	s.pullCmdMap.Store(seq, cmd)

	defer func() {
		if p := recover(); p != nil {
			Errorf("panic:\n%v\n%s", p, goutil.PanicTrace(1))
		}
	}()

	cmd.rerr = s.peer.pluginContainer.PreWritePull(cmd)
	if cmd.rerr != nil {
		cmd.done()
		return
	}

	var err error
W:
	if err = s.write(output); err != nil {
		if err == ErrConnClosed && s.redialForClient() {
			s.pullCmdMap.Delete(seq)
			s.seqLock.Lock()
			seq = s.seq
			s.seq++
			s.seqLock.Unlock()
			output.SetSeq(seq)
			s.pullCmdMap.Store(seq, cmd)
			goto W
		}
		cmd.rerr = rerror_writeFailed.Copy()
		cmd.rerr.Detail = err.Error()
		cmd.done()
		return
	}

	s.peer.pluginContainer.PostWritePull(cmd)
}

// Pull sends a packet and receives reply.
// Note:
// If the args is []byte or *[]byte type, it can automatically fill in the body codec name;
// If the session is a client role and PeerConfig.RedialTimes>0, it is automatically re-called once after a failure.
func (s *session) Pull(uri string, args interface{}, reply interface{}, setting ...socket.PacketSetting) *PullCmd {
	doneChan := make(chan *PullCmd, 1)
	s.AsyncPull(uri, args, reply, doneChan, setting...)
	pullCmd := <-doneChan
	close(doneChan)
	return pullCmd
}

// Push sends a packet, but do not receives reply.
// Note:
// If the args is []byte or *[]byte type, it can automatically fill in the body codec name;
// If the session is a client role and PeerConfig.RedialTimes>0, it is automatically re-called once after a failure.
func (s *session) Push(uri string, args interface{}, setting ...socket.PacketSetting) *Rerror {
	ctx := s.peer.getContext(s, true)
	ctx.start = s.peer.timeNow()
	output := ctx.output

	s.seqLock.Lock()
	seq := s.seq
	s.seq++
	s.seqLock.Unlock()

	output.SetSeq(seq)
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

	var err error
W:
	if err = s.write(output); err != nil {
		if err == ErrConnClosed && s.redialForClient() {
			s.seqLock.Lock()
			output.SetSeq(s.seq)
			s.seq++
			s.seqLock.Unlock()
			goto W
		}
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
	}()

	// read pull, pull reple or push
	for s.goonRead() {
		var ctx = s.peer.getContext(s, false)
		if s.peer.pluginContainer.PreReadHeader(ctx) != nil {
			s.peer.putContext(ctx, false)
			return
		}
		if readTimeout = s.ReadTimeout(); readTimeout > 0 {
			s.socket.SetReadDeadline(coarsetime.CeilingTimeNow().Add(readTimeout))
		}
		err = s.socket.ReadPacket(ctx.input)
		if err != nil || !s.goonRead() {
			s.peer.putContext(ctx, false)
			// DEBUG:
			// Debugf("s.socket.ReadPacket(ctx.input): err: %v", err)
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
	var (
		writeTimeout = s.WriteTimeout()
		now          time.Time
	)
	if writeTimeout > 0 {
		now = coarsetime.CeilingTimeNow()
	}

	s.writeLock.Lock()
	if status := s.getStatus(); status != statusOk {
		if !(status == statusActiveClosing && packet.Ptype() == TypeReply) {
			s.writeLock.Unlock()
			return ErrConnClosed
		}
	}

	defer func() {
		s.writeLock.Unlock()
		if err == io.EOF || err == socket.ErrProactivelyCloseSocket {
			err = ErrConnClosed
			if s.redialForClientFunc != nil {
				// Wait for the status to change
			W:
				if s.Health() {
					time.Sleep(time.Millisecond)
					goto W
				}
			}
		}
	}()

	if writeTimeout > 0 {
		s.socket.SetWriteDeadline(now.Add(writeTimeout))
	}
	err = s.socket.WritePacket(packet)
	return err
}

const (
	statusOk            int32 = 0
	statusActiveClosing int32 = 1
	statusActiveClosed  int32 = 2
	statusPassiveClosed int32 = 3
)

// Health checks if the session is ok.
func (s *session) Health() bool {
	return atomic.LoadInt32(&s.status) == statusOk
}

func (s *session) goonRead() bool {
	status := atomic.LoadInt32(&s.status)
	return status == statusOk || status == statusActiveClosing
}

// IsActiveClosed returns whether the connection has been closed, and is actively closed.
func (s *session) IsActiveClosed() bool {
	return atomic.LoadInt32(&s.status) == statusActiveClosed
}

func (s *session) activelyClosed() {
	atomic.StoreInt32(&s.status, statusActiveClosed)
}

func (s *session) activelyClosing() {
	atomic.StoreInt32(&s.status, statusActiveClosing)
}

// IsPassiveClosed returns whether the connection has been closed, and is passively closed.
func (s *session) IsPassiveClosed() bool {
	return atomic.LoadInt32(&s.status) == statusPassiveClosed
}

func (s *session) passivelyClosed() {
	atomic.StoreInt32(&s.status, statusPassiveClosed)
}

func (s *session) getStatus() int32 {
	return atomic.LoadInt32(&s.status)
}

// Close closes the session.
func (s *session) Close() error {
	s.closeLock.Lock()
	defer s.closeLock.Unlock()
	status := s.getStatus()
	if status != statusOk {
		return nil
	}

	s.activelyClosing()

	s.graceCtxWaitGroup.Wait()
	s.gracePullCmdWaitGroup.Wait()

	// Notice actively closed
	if !s.IsPassiveClosed() {
		s.activelyClosed()
	}

	err := s.socket.Close()
	s.peer.sessHub.Delete(s.Id())

	s.peer.pluginContainer.PostDisconnect(s)
	return err
}

func (s *session) readDisconnected(err error) {
	status := s.getStatus()
	if status == statusActiveClosed {
		return
	}
	// Notice passively closed
	s.passivelyClosed()

	if err != nil && err != io.EOF && err != socket.ErrProactivelyCloseSocket {
		Debugf("disconnect(%s) when reading: %s", s.RemoteIp(), err.Error())
	}
	s.graceCtxWaitGroup.Wait()

	if s.redialForClientFunc == nil || status == statusActiveClosing {
		s.pullCmdMap.Range(func(_, v interface{}) bool {
			pullCmd := v.(*PullCmd)
			pullCmd.cancel()
			return true
		})
	}

	if status == statusActiveClosing {
		return
	}

	s.socket.Close()
	s.peer.sessHub.Delete(s.Id())
	s.peer.pluginContainer.PostDisconnect(s)

	if !s.redialForClient() {
		s.pullCmdMap.Range(func(_, v interface{}) bool {
			pullCmd := v.(*PullCmd)
			pullCmd.cancel()
			return true
		})
	}
}

func (s *session) redialForClient() bool {
	if s.redialForClientFunc == nil {
		return false
	}
	s.redialLock.Lock()
	defer s.redialLock.Unlock()
	status := s.getStatus()
	if status == statusOk || status == statusActiveClosed || status == statusActiveClosing {
		return true
	}
	return s.redialForClientFunc()
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
