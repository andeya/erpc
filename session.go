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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/coarsetime"
	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/socket"
)

type (
	// PreSession a connection session that has not started reading goroutine.
	PreSession interface {
		// Peer returns the peer.
		Peer() Peer
		// LocalAddr returns the local network address.
		LocalAddr() net.Addr
		// RemoteAddr returns the remote network address.
		RemoteAddr() net.Addr
		// Public returns temporary public data of session(socket).
		Public() goutil.Map
		// PublicLen returns the length of public data of session(socket).
		PublicLen() int
		// SetId sets the session id.
		SetId(newId string)
		// ControlFD invokes f on the underlying connection's file
		// descriptor or handle.
		// The file descriptor fd is guaranteed to remain valid while
		// f executes but not after f returns.
		ControlFD(f func(fd uintptr)) error
		// ModifySocket modifies the socket.
		// Note:
		// The connection fd is not allowed to change!
		// Inherit the previous session id and temporary public data;
		// If modifiedConn!=nil, reset the net.Conn of the socket;
		// If newProtoFunc!=nil, reset the socket.ProtoFunc of the socket.
		ModifySocket(fn func(conn net.Conn) (modifiedConn net.Conn, newProtoFunc socket.ProtoFunc))
		// GetProtoFunc returns the socket.ProtoFunc
		GetProtoFunc() socket.ProtoFunc
		// Send sends packet to peer, before the formal connection.
		// Note:
		// the external setting seq is invalid, the internal will be forced to set;
		// does not support automatic redial after disconnection.
		Send(uri string, body interface{}, rerr *Rerror, setting ...socket.PacketSetting) *Rerror
		// Receive receives a packet from peer, before the formal connection.
		// Note: does not support automatic redial after disconnection.
		Receive(socket.NewBodyFunc, ...socket.PacketSetting) (*socket.Packet, *Rerror)
		// SessionAge returns the session max age.
		SessionAge() time.Duration
		// ContextAge returns PULL or PUSH context max age.
		ContextAge() time.Duration
		// SetSessionAge sets the session max age.
		SetSessionAge(duration time.Duration)
		// SetContextAge sets PULL or PUSH context max age.
		SetContextAge(duration time.Duration)
	}
	// BaseSession a connection session with the common method set.
	BaseSession interface {
		// Id returns the session id.
		Id() string
		// Peer returns the peer.
		Peer() Peer
		// LocalAddr returns the local network address.
		LocalAddr() net.Addr
		// RemoteAddr returns the remote network address.
		RemoteAddr() net.Addr
		// Public returns temporary public data of session(socket).
		Public() goutil.Map
		// PublicLen returns the length of public data of session(socket).
		PublicLen() int
	}
	// Session a connection session.
	Session interface {
		BaseSession
		// SetId sets the session id.
		SetId(newId string)
		// Close closes the session.
		Close() error
		// Health checks if the session is usable.
		Health() bool
		// AsyncPull sends a packet and receives reply asynchronously.
		// If the args is []byte or *[]byte type, it can automatically fill in the body codec name.
		AsyncPull(
			uri string,
			args interface{},
			reply interface{},
			pullCmdChan chan<- PullCmd,
			setting ...socket.PacketSetting,
		) PullCmd
		// Pull sends a packet and receives reply.
		// Note:
		// If the args is []byte or *[]byte type, it can automatically fill in the body codec name;
		// If the session is a client role and PeerConfig.RedialTimes>0, it is automatically re-called once after a failure.
		Pull(uri string, args interface{}, reply interface{}, setting ...socket.PacketSetting) PullCmd
		// Push sends a packet, but do not receives reply.
		// Note:
		// If the args is []byte or *[]byte type, it can automatically fill in the body codec name;
		// If the session is a client role and PeerConfig.RedialTimes>0, it is automatically re-called once after a failure.
		Push(uri string, args interface{}, setting ...socket.PacketSetting) *Rerror
		// SessionAge returns the session max age.
		SessionAge() time.Duration
		// ContextAge returns PULL or PUSH context max age.
		ContextAge() time.Duration
	}

	session struct {
		peer                           *peer
		getPullHandler, getPushHandler func(uriPath string) (*Handler, bool)
		timeSince                      func(time.Time) time.Duration
		timeNow                        func() time.Time
		seq                            uint64
		seqLock                        sync.Mutex
		pullCmdMap                     goutil.Map
		protoFuncs                     []socket.ProtoFunc
		socket                         socket.Socket
		status                         int32 // 0:ok, 1:active closed, 2:disconnect
		statusLock                     sync.Mutex
		writeLock                      sync.Mutex
		graceCtxWaitGroup              sync.WaitGroup
		gracePullCmdWaitGroup          sync.WaitGroup
		sessionAge                     time.Duration
		contextAge                     time.Duration
		sessionAgeLock                 sync.RWMutex
		contextAgeLock                 sync.RWMutex
		conn                           net.Conn
		lock                           sync.RWMutex
		// only for client role
		redialForClientLocked func(oldConn net.Conn) bool
	}
)

var (
	_ PreSession  = new(session)
	_ Session     = new(session)
	_ BaseSession = new(session)
)

func newSession(peer *peer, conn net.Conn, protoFuncs []socket.ProtoFunc) *session {
	var s = &session{
		peer:           peer,
		getPullHandler: peer.router.subRouter.getPull,
		getPushHandler: peer.router.subRouter.getPush,
		timeSince:      peer.timeSince,
		timeNow:        peer.timeNow,
		conn:           conn,
		protoFuncs:     protoFuncs,
		socket:         socket.NewSocket(conn, protoFuncs...),
		pullCmdMap:     goutil.AtomicMap(),
		sessionAge:     peer.defaultSessionAge,
		contextAge:     peer.defaultContextAge,
	}
	return s
}

// Peer returns the peer.
func (s *session) Peer() Peer {
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

// ControlFD invokes f on the underlying connection's file
// descriptor or handle.
// The file descriptor fd is guaranteed to remain valid while
// f executes but not after f returns.
func (s *session) ControlFD(f func(fd uintptr)) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.socket.ControlFD(f)
}

func (s *session) getConn() net.Conn {
	s.lock.RLock()
	c := s.conn
	s.lock.RUnlock()
	return c
}

// ModifySocket modifies the socket.
// Note:
// The connection fd is not allowed to change!
// Inherit the previous session id and temporary public data;
// If modifiedConn!=nil, reset the net.Conn of the socket;
// If newProtoFunc!=nil, reset the socket.ProtoFunc of the socket.
func (s *session) ModifySocket(fn func(conn net.Conn) (modifiedConn net.Conn, newProtoFunc socket.ProtoFunc)) {
	s.lock.Lock()
	defer s.lock.Unlock()
	modifiedConn, newProtoFunc := fn(s.conn)
	isModifiedConn := modifiedConn != nil
	if isModifiedConn {
		s.conn = modifiedConn
	}
	isNewProtoFunc := newProtoFunc != nil
	if isNewProtoFunc {
		s.protoFuncs = s.protoFuncs[:0]
		s.protoFuncs = append(s.protoFuncs, newProtoFunc)
	}
	if !isModifiedConn && !isNewProtoFunc {
		return
	}
	pub := s.socket.Public()
	id := s.Id()
	s.socket = socket.NewSocket(s.conn, s.protoFuncs...)
	nebPub := s.socket.Public()
	pub.Range(func(key, value interface{}) bool {
		nebPub.Store(key, value)
		return true
	})
	s.socket.SetId(id)
}

// GetProtoFunc returns the socket.ProtoFunc
func (s *session) GetProtoFunc() socket.ProtoFunc {
	if len(s.protoFuncs) > 0 && s.protoFuncs[0] != nil {
		return s.protoFuncs[0]
	}
	return socket.DefaultProtoFunc()
}

// LocalAddr returns the local network address.
func (s *session) LocalAddr() net.Addr {
	return s.socket.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (s *session) RemoteAddr() net.Addr {
	return s.socket.RemoteAddr()
}

// SessionAge returns the session max age.
func (s *session) SessionAge() time.Duration {
	s.sessionAgeLock.RLock()
	age := s.sessionAge
	s.sessionAgeLock.RUnlock()
	return age
}

// SetSessionAge sets the session max age.
func (s *session) SetSessionAge(duration time.Duration) {
	s.sessionAgeLock.Lock()
	s.sessionAge = duration
	if duration > 0 {
		s.socket.SetReadDeadline(coarsetime.CeilingTimeNow().Add(duration))
	} else {
		s.socket.SetReadDeadline(time.Time{})
	}
	s.sessionAgeLock.Unlock()
}

// ContextAge returns PULL or PUSH context max age.
func (s *session) ContextAge() time.Duration {
	s.contextAgeLock.RLock()
	age := s.contextAge
	s.contextAgeLock.RUnlock()
	return age
}

// SetContextAge sets PULL or PUSH context max age.
func (s *session) SetContextAge(duration time.Duration) {
	s.contextAgeLock.Lock()
	s.contextAge = duration
	s.contextAgeLock.Unlock()
}

// Send sends packet to peer, before the formal connection.
// Note:
// the external setting seq is invalid, the internal will be forced to set;
// does not support automatic redial after disconnection.
func (s *session) Send(uri string, body interface{}, rerr *Rerror, setting ...socket.PacketSetting) *Rerror {
	output := socket.GetPacket(setting...)
	s.seqLock.Lock()
	output.SetSeq(s.seq)
	s.seq++
	s.seqLock.Unlock()
	if output.BodyCodec() == codec.NilCodecId {
		output.SetBodyCodec(s.peer.defaultBodyCodec)
	}
	if len(uri) > 0 {
		output.SetUri(uri)
	}
	if body != nil {
		output.SetBody(body)
	}
	if rerr != nil {
		rerr.SetToMeta(output.Meta())
	}
	if age := s.ContextAge(); age > 0 {
		ctxTimout, _ := context.WithTimeout(output.Context(), age)
		socket.WithContext(ctxTimout)(output)
	}
	_, rerr = s.write(output)
	socket.PutPacket(output)
	return rerr
}

// Receive receives a packet from peer, before the formal connection.
// Note: does not support automatic redial after disconnection.
func (s *session) Receive(newBodyFunc socket.NewBodyFunc, setting ...socket.PacketSetting) (*socket.Packet, *Rerror) {
	input := socket.GetPacket(setting...)
	input.SetNewBody(newBodyFunc)

	if age := s.ContextAge(); age > 0 {
		ctxTimout, _ := context.WithTimeout(input.Context(), age)
		socket.WithContext(ctxTimout)(input)
	}
	deadline, _ := input.Context().Deadline()
	s.socket.SetReadDeadline(deadline)

	if err := s.socket.ReadPacket(input); err != nil {
		rerr := rerrConnClosed.Copy()
		rerr.Detail = err.Error()
		socket.PutPacket(input)
		return nil, rerr
	}
	rerr := NewRerrorFromMeta(input.Meta())
	return input, rerr
}

// AsyncPull sends a packet and receives reply asynchronously.
// Note:
// If the args is []byte or *[]byte type, it can automatically fill in the body codec name;
// If the session is a client role and PeerConfig.RedialTimes>0, it is automatically re-called once after a failure.
func (s *session) AsyncPull(
	uri string,
	args interface{},
	reply interface{},
	pullCmdChan chan<- PullCmd,
	setting ...socket.PacketSetting,
) PullCmd {
	if pullCmdChan == nil {
		pullCmdChan = make(chan PullCmd, 10) // buffered.
	} else {
		// If caller passes pullCmdChan != nil, it must arrange that
		// pullCmdChan has enough buffer for the number of simultaneous
		// RPCs that will be using that channel. If the channel
		// is totally unbuffered, it's best not to run at all.
		if cap(pullCmdChan) == 0 {
			Panicf("*session.AsyncPull(): pullCmdChan channel is unbuffered")
		}
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
		if fn != nil {
			fn(output)
		}
	}
	if output.BodyCodec() == codec.NilCodecId {
		output.SetBodyCodec(s.peer.defaultBodyCodec)
	}
	if age := s.ContextAge(); age > 0 {
		ctxTimout, _ := context.WithTimeout(output.Context(), age)
		socket.WithContext(ctxTimout)(output)
	}

	cmd := &pullCmd{
		sess:        s,
		output:      output,
		reply:       reply,
		pullCmdChan: pullCmdChan,
		doneChan:    make(chan struct{}),
		start:       s.peer.timeNow(),
		public:      goutil.RwMap(),
	}

	// count pull-launch
	s.gracePullCmdWaitGroup.Add(1)

	if s.socket.PublicLen() > 0 {
		s.socket.Public().Range(func(key, value interface{}) bool {
			cmd.public.Store(key, value)
			return true
		})
	}

	cmd.mu.Lock()
	defer cmd.mu.Unlock()

	s.pullCmdMap.Store(seq, cmd)

	defer func() {
		if p := recover(); p != nil {
			Errorf("panic:\n%v\n%s", p, goutil.PanicTrace(1))
		}
	}()

	cmd.rerr = s.peer.pluginContainer.PreWritePull(cmd)
	if cmd.rerr != nil {
		cmd.done()
		return cmd
	}
	var usedConn net.Conn
W:
	if usedConn, cmd.rerr = s.write(output); cmd.rerr != nil {
		if cmd.rerr == rerrConnClosed && s.redialForClient(usedConn) {
			goto W
		}
		cmd.done()
		return cmd
	}

	s.peer.pluginContainer.PostWritePull(cmd)
	return cmd
}

// Pull sends a packet and receives reply.
// Note:
// If the args is []byte or *[]byte type, it can automatically fill in the body codec name;
// If the session is a client role and PeerConfig.RedialTimes>0, it is automatically re-called once after a failure.
func (s *session) Pull(uri string, args interface{}, reply interface{}, setting ...socket.PacketSetting) PullCmd {
	pullCmd := s.AsyncPull(uri, args, reply, make(chan PullCmd, 1), setting...)
	<-pullCmd.Done()
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
		if fn != nil {
			fn(output)
		}
	}
	if output.BodyCodec() == codec.NilCodecId {
		output.SetBodyCodec(s.peer.defaultBodyCodec)
	}
	if age := s.ContextAge(); age > 0 {
		ctxTimout, _ := context.WithTimeout(output.Context(), age)
		socket.WithContext(ctxTimout)(output)
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

	var usedConn net.Conn
W:
	if usedConn, rerr = s.write(output); rerr != nil {
		if rerr == rerrConnClosed && s.redialForClient(usedConn) {
			goto W
		}
		return rerr
	}

	s.runlog("", s.peer.timeSince(ctx.start), nil, output, typePushLaunch)
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

const (
	statusOk            int32 = 0
	statusActiveClosing int32 = 1
	statusActiveClosed  int32 = 2
	statusPassiveClosed int32 = 3
)

// Health checks if the session is usable.
func (s *session) Health() bool {
	status := s.getStatus()
	if status == statusOk {
		return true
	}
	if s.redialForClientLocked == nil {
		return false
	}
	if status == statusPassiveClosed {
		return true
	}
	return false
}

// isOk checks if the session is normal.
func (s *session) isOk() bool {
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
	s.lock.Lock()

	s.statusLock.Lock()
	status := s.getStatus()
	if status != statusOk {
		s.statusLock.Unlock()
		s.lock.Unlock()
		return nil
	}
	s.activelyClosing()
	s.statusLock.Unlock()

	s.peer.sessHub.Delete(s.Id())

	s.graceCtxWaitGroup.Wait()
	s.gracePullCmdWaitGroup.Wait()

	s.statusLock.Lock()
	// Notice actively closed
	if !s.IsPassiveClosed() {
		s.activelyClosed()
	}
	s.statusLock.Unlock()

	err := s.socket.Close()
	s.lock.Unlock()

	s.peer.pluginContainer.PostDisconnect(s)
	return err
}

func (s *session) readDisconnected(oldConn net.Conn, err error) {
	s.statusLock.Lock()
	status := s.getStatus()
	if status == statusActiveClosed {
		s.statusLock.Unlock()
		return
	}
	// Notice passively closed
	s.passivelyClosed()
	s.statusLock.Unlock()

	s.peer.sessHub.Delete(s.Id())

	if err != nil && err != io.EOF && err != socket.ErrProactivelyCloseSocket {
		Debugf("disconnect(%s) when reading: %s", s.RemoteAddr().String(), err.Error())
	}
	s.graceCtxWaitGroup.Wait()

	// cancel the pullCmd that is waiting for a reply
	s.pullCmdMap.Range(func(_, v interface{}) bool {
		pullCmd := v.(*pullCmd)
		pullCmd.mu.Lock()
		if !pullCmd.hasReply() && pullCmd.rerr == nil {
			pullCmd.cancel()
		}
		pullCmd.mu.Unlock()
		return true
	})

	if status == statusActiveClosing {
		return
	}

	s.socket.Close()

	if !s.redialForClient(oldConn) {
		s.peer.pluginContainer.PostDisconnect(s)
	}
}

func (s *session) redialForClient(oldConn net.Conn) bool {
	if s.redialForClientLocked == nil {
		return false
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	status := s.getStatus()
	if status == statusActiveClosed || status == statusActiveClosing {
		return false
	}
	return s.redialForClientLocked(oldConn)
}

func (s *session) startReadAndHandle() {
	var withContext socket.PacketSetting
	if readTimeout := s.SessionAge(); readTimeout > 0 {
		s.socket.SetReadDeadline(coarsetime.CeilingTimeNow().Add(readTimeout))
		ctxTimout, _ := context.WithTimeout(context.Background(), readTimeout)
		withContext = socket.WithContext(ctxTimout)
	} else {
		s.socket.SetReadDeadline(time.Time{})
		withContext = socket.WithContext(nil)
	}

	var (
		err  error
		conn = s.getConn()
	)
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%v\n%s", p, goutil.PanicTrace(2))
		}
		s.readDisconnected(conn, err)
	}()

	// read pull, pull reple or push
	for s.goonRead() {
		var ctx = s.peer.getContext(s, false)
		withContext(ctx.input)
		if s.peer.pluginContainer.PreReadHeader(ctx) != nil {
			s.peer.putContext(ctx, false)
			return
		}
		err = s.socket.ReadPacket(ctx.input)
		if err != nil || !s.goonRead() {
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

func (s *session) write(packet *socket.Packet) (net.Conn, *Rerror) {
	conn := s.getConn()
	status := s.getStatus()
	if status != statusOk &&
		!(status == statusActiveClosing && packet.Ptype() == TypeReply) {
		return conn, rerrConnClosed
	}

	var (
		rerr        *Rerror
		err         error
		ctx         = packet.Context()
		deadline, _ = ctx.Deadline()
	)
	select {
	case <-ctx.Done():
		err = ctx.Err()
		goto ERR
	default:
	}

	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	select {
	case <-ctx.Done():
		err = ctx.Err()
		goto ERR
	default:
		s.socket.SetWriteDeadline(deadline)
		err = s.socket.WritePacket(packet)
	}

	if err == nil {
		return conn, nil
	}

	if err == io.EOF || err == socket.ErrProactivelyCloseSocket {
		return conn, rerrConnClosed
	}

	Debugf("write error: %s", err.Error())

ERR:
	rerr = rerrWriteFailed.Copy()
	rerr.Detail = err.Error()
	return conn, rerr
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

func (s *session) runlog(realIp string, costTime time.Duration, input, output *socket.Packet, logType int8) {
	var addr = s.RemoteAddr().String()
	if realIp != "" && realIp != addr {
		addr += "(real: " + realIp + ")"
	}
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
				const logformat = "[push-launch] remote: %s | cost-time: %s%s | uri: %-30s |\nSEND:\n size: %d\n body[-json]: %s\n"
				printFunc(logformat, addr, costTime, slowStr, output.Uri(), output.Size(), bodyLogBytes(output))

			} else {
				const logformat = "[push-launch] remote: %s | cost-time: %s%s | uri: %-30s |\nSEND:\n size: %d\n"
				printFunc(logformat, addr, costTime, slowStr, output.Uri(), output.Size())
			}

		case typePushHandle:
			if s.peer.printBody {
				const logformat = "[push-handle] remote: %s | cost-time: %s%s | uri: %-30s |\nRECV:\n size: %d\n body[-json]: %s\n"
				printFunc(logformat, addr, costTime, slowStr, input.Uri(), input.Size(), bodyLogBytes(input))
			} else {
				const logformat = "[push-handle] remote: %s | cost-time: %s%s | uri: %-30s |\nRECV:\n size: %d\n"
				printFunc(logformat, addr, costTime, slowStr, input.Uri(), input.Size())
			}

		case typePullLaunch:
			if s.peer.printBody {
				const logformat = "[pull-launch] remote: %s | cost-time: %s%s | uri: %-30s |\nSEND:\n size: %d\n body[-json]: %s\nRECV:\n size: %d\n status: %s\n body[-json]: %s\n"
				printFunc(logformat, addr, costTime, slowStr, output.Uri(), output.Size(), bodyLogBytes(output), input.Size(), getRerrorBytes(input.Meta()), bodyLogBytes(input))
			} else {
				const logformat = "[pull-launch] remote: %s | cost-time: %s%s | uri: %-30s |\nSEND:\n size: %d\nRECV:\n size: %d\n status: %s\n"
				printFunc(logformat, addr, costTime, slowStr, output.Uri(), output.Size(), input.Size(), getRerrorBytes(input.Meta()))
			}

		case typePullHandle:
			if s.peer.printBody {
				const logformat = "[pull-handle] remote: %s | cost-time: %s%s | uri: %-30s |\nRECV:\n size: %d\n body[-json]: %s\nSEND:\n size: %d\n status: %s\n body[-json]: %s\n"
				printFunc(logformat, addr, costTime, slowStr, input.Uri(), input.Size(), bodyLogBytes(input), output.Size(), getRerrorBytes(output.Meta()), bodyLogBytes(output))
			} else {
				const logformat = "[pull-handle] remote: %s | cost-time: %s%s | uri: %-30s |\nRECV:\n size: %d\nSEND:\n size: %d\n status: %s\n"
				printFunc(logformat, addr, costTime, slowStr, input.Uri(), input.Size(), output.Size(), getRerrorBytes(output.Meta()))
			}
		}
	} else {
		switch logType {
		case typePushLaunch:
			if s.peer.printBody {
				const logformat = "[push-launch] remote: %s | uri: %-30s |\nSEND:\n size: %d\n body[-json]: %s\n"
				Infof(logformat, addr, output.Uri(), output.Size(), bodyLogBytes(output))

			} else {
				const logformat = "[push-launch] remote: %s | uri: %-30s |\nSEND:\n size: %d\n"
				Infof(logformat, addr, output.Uri(), output.Size())
			}

		case typePushHandle:
			if s.peer.printBody {
				const logformat = "[push-handle] remote: %s | uri: %-30s |\nRECV:\n size: %d\n body[-json]: %s\n"
				Infof(logformat, addr, input.Uri(), input.Size(), bodyLogBytes(input))
			} else {
				const logformat = "[push-handle] remote: %s | uri: %-30s |\nRECV:\n size: %d\n"
				Infof(logformat, addr, input.Uri(), input.Size())
			}

		case typePullLaunch:
			if s.peer.printBody {
				const logformat = "[pull-launch] remote: %s | uri: %-30s |\nSEND:\n size: %d\n body[-json]: %s\nRECV:\n size: %d\n status: %s\n body[-json]: %s\n"
				Infof(logformat, addr, output.Uri(), output.Size(), bodyLogBytes(output), input.Size(), getRerrorBytes(input.Meta()), bodyLogBytes(input))
			} else {
				const logformat = "[pull-launch] remote: %s | uri: %-30s |\nSEND:\n size: %d\nRECV:\n size: %d\n status: %s\n"
				Infof(logformat, addr, output.Uri(), output.Size(), input.Size(), getRerrorBytes(input.Meta()))
			}

		case typePullHandle:
			if s.peer.printBody {
				const logformat = "[pull-handle] remote: %s | uri: %-30s |\nRECV:\n size: %d\n body[-json]: %s\nSEND:\n size: %d\n status: %s\n body[-json]: %s\n"
				Infof(logformat, addr, input.Uri(), input.Size(), bodyLogBytes(input), output.Size(), getRerrorBytes(output.Meta()), bodyLogBytes(output))
			} else {
				const logformat = "[pull-handle] remote: %s | uri: %-30s |\nRECV:\n size: %d\nSEND:\n size: %d\n status: %s\n"
				Infof(logformat, addr, input.Uri(), input.Size(), output.Size(), getRerrorBytes(output.Meta()))
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
