// Copyright 2015-2018 HenryLee. All Rights Reserved.
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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/coarsetime"
	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/socket"
	"github.com/henrylee2cn/teleport/utils"
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
		// Swap returns custom data swap of the session(socket).
		Swap() goutil.Map
		// SetID sets the session id.
		SetID(newID string)
		// ControlFD invokes f on the underlying connection's file
		// descriptor or handle.
		// The file descriptor fd is guaranteed to remain valid while
		// f executes but not after f returns.
		ControlFD(f func(fd uintptr)) error
		// ModifySocket modifies the socket.
		// NOTE:
		// The connection fd is not allowed to change!
		// Inherit the previous session id and custom data swap;
		// If modifiedConn!=nil, reset the net.Conn of the socket;
		// If newProtoFunc!=nil, reset the ProtoFunc of the socket.
		ModifySocket(fn func(conn net.Conn) (modifiedConn net.Conn, newProtoFunc ProtoFunc))
		// GetProtoFunc returns the ProtoFunc
		GetProtoFunc() ProtoFunc
		// PreSend temporarily sends message when the session is just builded,
		// do not execute other plugins.
		// NOTE:
		//  Cannot be called during the Non-PostDial and Non-PostAccept phase;
		//  Does not support automatic redial after disconnection;
		//  Recommend to reuse unused Message: PutMessage(input).
		PreSend(mtype byte, serviceMethod string, body interface{}, stat *Status, setting ...MessageSetting) (opStat *Status)
		// PreReceive temporarily receives message when the session is just builded,
		// do not execute other plugins.
		// NOTE:
		//  Cannot be called during the Non-PostDial and Non-PostAccept phase;
		//  Does not support automatic redial after disconnection;
		//  Recommend to reuse unused Message: PutMessage(input).
		PreReceive(newArgs NewBodyFunc, ctx ...context.Context) (input Message)
		// PreCall temporarily sends TypeCall message and receives message,
		// when the session is just builded, do not execute other plugins.
		// NOTE:
		//  Cannot be called during the Non-PostDial and Non-PostAccept phase;
		//  The reply parameter is the body receiver;
		//  The external setting seq is invalid, the internal will be forced to set;
		//  Does not support automatic redial after disconnection.
		PreCall(serviceMethod string, args, reply interface{}, callSetting ...MessageSetting) (opStat *Status)
		// PreReply temporarily sends TypeReply message when the session is just builded,
		// do not execute other plugins.
		// NOTE:
		//  Cannot be called during the Non-PostDial and Non-PostAccept phase;
		//  The external setting seq is invalid, the internal will be forced to set;
		//  Does not support automatic redial after disconnection.
		PreReply(req Message, body interface{}, stat *Status, setting ...MessageSetting) (opStat *Status)
		// RawPush sends a TypePush message without executing other plugins.
		// NOTE:
		//  The external setting seq is invalid, the internal will be forced to set;
		//  Does not support automatic redial after disconnection.
		RawPush(serviceMethod string, args interface{}, setting ...MessageSetting) (opStat *Status)
		// SessionAge returns the session max age.
		SessionAge() time.Duration
		// ContextAge returns CALL or PUSH context max age.
		ContextAge() time.Duration
		// SetSessionAge sets the session max age.
		SetSessionAge(duration time.Duration)
		// SetContextAge sets CALL or PUSH context max age.
		SetContextAge(duration time.Duration)
		// Logger logger interface
		Logger
	}
	// BaseSession a connection session with the common method set.
	BaseSession interface {
		// Peer returns the peer.
		Peer() Peer
		// ID returns the session id.
		ID() string
		// LocalAddr returns the local network address.
		LocalAddr() net.Addr
		// RemoteAddr returns the remote network address.
		RemoteAddr() net.Addr
		// Swap returns custom data swap of the session(socket).
		Swap() goutil.Map
		// Logger logger interface
		Logger
	}
	// CtxSession a connection session that can be used in the handler context.
	CtxSession interface {
		// ID returns the session id.
		ID() string
		// LocalAddr returns the local network address.
		LocalAddr() net.Addr
		// RemoteAddr returns the remote network address.
		RemoteAddr() net.Addr
		// Swap returns custom data swap of the session(socket).
		Swap() goutil.Map
		// CloseNotify returns a channel that closes when the connection has gone away.
		CloseNotify() <-chan struct{}
		// Health checks if the session is usable.
		Health() bool
		// AsyncCall sends a message and receives reply asynchronously.
		// If the  is []byte or *[]byte type, it can automatically fill in the body codec name.
		AsyncCall(
			serviceMethod string,
			args interface{},
			result interface{},
			callCmdChan chan<- CallCmd,
			setting ...MessageSetting,
		) CallCmd
		// Call sends a message and receives reply.
		// NOTE:
		// If the args is []byte or *[]byte type, it can automatically fill in the body codec name;
		// If the session is a client role and PeerConfig.RedialTimes>0, it is automatically re-called once after a failure.
		Call(serviceMethod string, args interface{}, result interface{}, setting ...MessageSetting) CallCmd
		// Push sends a message of TypePush type, but do not receives reply.
		// NOTE:
		// If the args is []byte or *[]byte type, it can automatically fill in the body codec name;
		// If the session is a client role and PeerConfig.RedialTimes>0, it is automatically re-called once after a failure.
		Push(serviceMethod string, args interface{}, setting ...MessageSetting) *Status
		// SessionAge returns the session max age.
		SessionAge() time.Duration
		// ContextAge returns CALL or PUSH context max age.
		ContextAge() time.Duration
		// Logger logger interface
		Logger
	}
	// Session a connection session.
	Session interface {
		// Peer returns the peer.
		Peer() Peer
		// SetID sets the session id.
		SetID(newID string)
		// Close closes the session.
		Close() error
		CtxSession
	}
)

var (
	_ PreSession  = new(session)
	_ BaseSession = new(session)
	_ CtxSession  = new(session)
	_ Session     = new(session)
)

type session struct {
	peer                           *peer
	getCallHandler, getPushHandler func(serviceMethodPath string) (*Handler, bool)
	timeNow                        func() int64
	callCmdMap                     goutil.Map
	protoFuncs                     []ProtoFunc
	socket                         socket.Socket
	closeNotifyCh                  chan struct{} // closeNotifyCh is the channel returned by CloseNotify.
	writeLock                      sync.Mutex
	graceCtxWaitGroup              sync.WaitGroup
	graceCtxMutex                  sync.Mutex
	graceCallCmdWaitGroup          sync.WaitGroup
	sessionAge                     time.Duration
	contextAge                     time.Duration
	sessionAgeLock                 sync.RWMutex
	contextAgeLock                 sync.RWMutex
	lock                           sync.RWMutex
	redialForClientLocked          func() bool // only for client role
	seq                            int32
	status                         int32
	didCloseNotify                 int32
}

func newSession(peer *peer, conn net.Conn, protoFuncs []ProtoFunc) *session {
	var s = &session{
		peer:           peer,
		getCallHandler: peer.router.subRouter.getCall,
		getPushHandler: peer.router.subRouter.getPush,
		timeNow:        peer.timeNow,
		protoFuncs:     protoFuncs,
		status:         statusPreparing,
		socket:         socket.NewSocket(conn, protoFuncs...),
		closeNotifyCh:  make(chan struct{}),
		callCmdMap:     goutil.AtomicMap(),
		sessionAge:     peer.defaultSessionAge,
		contextAge:     peer.defaultContextAge,
	}
	return s
}

// NOTE: Do not change the order
const (
	statusPreparing int32 = iota
	statusOk
	statusActiveClosing
	statusActiveClosed
	statusPassiveClosing
	statusPassiveClosed
	statusRedialing
	statusRedialFailed
)

func (s *session) changeStatus(stat int32) {
	atomic.StoreInt32(&s.status, stat)
}

func (s *session) tryChangeStatus(to int32, fromList ...int32) (changed bool) {
	for _, from := range fromList {
		if atomic.CompareAndSwapInt32(&s.status, from, to) {
			return true
		}
	}
	return false
}

func (s *session) checkStatus(checkList ...int32) bool {
	stat := atomic.LoadInt32(&s.status)
	for _, v := range checkList {
		if v == stat {
			return true
		}
	}
	return false
}

func (s *session) getStatus() int32 {
	return atomic.LoadInt32(&s.status)
}

func (s *session) goonRead() bool {
	return s.checkStatus(statusOk, statusActiveClosing)
}

func (s *session) notifyClosed() {
	if atomic.CompareAndSwapInt32(&s.didCloseNotify, 0, 1) {
		close(s.closeNotifyCh)
	}
}

// CloseNotify returns a channel that closes when the connection has gone away.
func (s *session) CloseNotify() <-chan struct{} {
	return s.closeNotifyCh
}

// IsActiveClosed returns whether the connection has been closed, and is actively closed.
func (s *session) IsActiveClosed() bool {
	return s.checkStatus(statusActiveClosed)
}

// IsPassiveClosed returns whether the connection has been closed, and is passively closed.
func (s *session) IsPassiveClosed() bool {
	return s.checkStatus(statusPassiveClosed)
}

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

func (s *session) graceCtxWait() {
	s.graceCtxMutex.Lock()
	s.graceCtxWaitGroup.Wait()
	s.graceCtxMutex.Unlock()
}

// Peer returns the peer.
func (s *session) Peer() Peer {
	return s.peer
}

// ID returns the session id.
func (s *session) ID() string {
	return s.socket.ID()
}

// SetID sets the session id.
func (s *session) SetID(newID string) {
	oldID := s.ID()
	if oldID == newID {
		return
	}
	s.socket.SetID(newID)
	hub := s.peer.sessHub
	hub.Set(s)
	hub.Delete(oldID)
	Tracef("session changes id: %s -> %s", oldID, newID)
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
	return s.socket.Raw()
}

// ModifySocket modifies the socket.
// NOTE:
// The connection fd is not allowed to change!
// Inherit the previous session id and custom data swap;
// If modifiedConn!=nil, reset the net.Conn of the socket;
// If newProtoFunc!=nil, reset the ProtoFunc of the socket.
func (s *session) ModifySocket(fn func(conn net.Conn) (modifiedConn net.Conn, newProtoFunc ProtoFunc)) {
	s.lock.Lock()
	defer s.lock.Unlock()
	modifiedConn, newProtoFunc := fn(s.getConn())
	isModifiedConn := modifiedConn != nil
	isNewProtoFunc := newProtoFunc != nil
	if isNewProtoFunc {
		s.protoFuncs = s.protoFuncs[:0]
		s.protoFuncs = append(s.protoFuncs, newProtoFunc)
	}
	if !isModifiedConn && !isNewProtoFunc {
		return
	}
	var (
		pub   goutil.Map
		count = s.socket.SwapLen()
		id    = s.ID()
	)
	if count > 0 {
		pub = s.socket.Swap()
	}
	s.socket = socket.NewSocket(modifiedConn, s.protoFuncs...)
	if count > 0 {
		newPub := s.socket.Swap()
		pub.Range(func(key, value interface{}) bool {
			newPub.Store(key, value)
			return true
		})
	}
	s.socket.SetID(id)
}

// GetProtoFunc returns the ProtoFunc
func (s *session) GetProtoFunc() ProtoFunc {
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

// ContextAge returns CALL or PUSH context max age.
func (s *session) ContextAge() time.Duration {
	s.contextAgeLock.RLock()
	age := s.contextAge
	s.contextAgeLock.RUnlock()
	return age
}

// SetContextAge sets CALL or PUSH context max age.
func (s *session) SetContextAge(duration time.Duration) {
	s.contextAgeLock.Lock()
	s.contextAge = duration
	s.contextAgeLock.Unlock()
}

// PreSend temporarily sends message when the session is just builded,
// do not execute other plugins.
// NOTE:
//  Cannot be called during the Non-PostDial and Non-PostAccept phase;
//  Does not support automatic redial after disconnection;
//  Recommend to reuse unused Message: PutMessage(input).
func (s *session) PreSend(mtype byte, serviceMethod string, body interface{}, stat *Status, setting ...MessageSetting) (opStat *Status) {
	if !s.checkStatus(statusPreparing) {
		return statUnpreparedError
	}
	var output Message
	defer func() {
		if output != nil {
			socket.PutMessage(output)
		}
		if p := recover(); p != nil {
			opStat = statBadMessage.Copy(p, 3)
		}
	}()
	output, opStat = s.send(mtype, 0, serviceMethod, body, stat, setting)
	return opStat
}

func (s *session) send(mtype byte, seq int32, serviceMethod string, body interface{}, stat *Status, setting []MessageSetting) (Message, *Status) {
	output := socket.GetMessage(setting...)
	output.SetMtype(mtype)
	if seq == 0 {
		seq = atomic.AddInt32(&s.seq, 1)
	}
	output.SetSeq(seq)
	if output.BodyCodec() == codec.NilCodecID {
		output.SetBodyCodec(s.peer.defaultBodyCodec)
	}
	if len(serviceMethod) > 0 {
		output.SetServiceMethod(serviceMethod)
	}
	if body != nil {
		output.SetBody(body)
	}
	if !stat.OK() {
		output.SetStatus(stat)
	}
	return output, s.doSend(output)
}

func (s *session) doSend(output Message) *Status {
	if age := s.ContextAge(); age > 0 {
		ctxTimout, _ := context.WithTimeout(output.Context(), age)
		socket.WithContext(ctxTimout)(output)
	}

	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	ctx := output.Context()
	select {
	case <-ctx.Done():
		return statWriteFailed.Copy(ctx.Err())
	default:
		deadline, _ := ctx.Deadline()
		s.socket.SetWriteDeadline(deadline)
		err := s.socket.WriteMessage(output)
		if err == nil {
			return nil
		}
		if err == io.EOF || err == socket.ErrProactivelyCloseSocket {
			return statConnClosed
		}
		Debugf("write error: %s", err.Error())
		return statWriteFailed.Copy(err)
	}
}

var statUnpreparedError = statInvalidOpError.Copy("Cannot be called during the Non-PostDial and Non-PostAccept phase")

// PreReceive temporarily receives message when the session is just builded,
// do not execute other plugins.
// NOTE:
//  Cannot be called during the Non-PostDial and Non-PostAccept phase;
//  Does not support automatic redial after disconnection;
//  Recommend to reuse unused Message: PutMessage(input).
func (s *session) PreReceive(newArgs NewBodyFunc, ctx ...context.Context) (input Message) {
	if len(ctx) > 0 {
		input = socket.GetMessage(WithContext(ctx[0]))
	} else {
		input = socket.GetMessage()
	}
	if !s.checkStatus(statusPreparing) {
		input.SetStatus(statUnpreparedError)
		return input
	}
	input.SetNewBody(newArgs)
	defer func() {
		if p := recover(); p != nil {
			input.SetStatus(statBadMessage.Copy(p, 3))
		}
	}()

	if age := s.ContextAge(); age > 0 {
		ctxTimout, _ := context.WithTimeout(input.Context(), age)
		socket.WithContext(ctxTimout)(input)
	}
	deadline, _ := input.Context().Deadline()
	s.socket.SetReadDeadline(deadline)

	if err := s.socket.ReadMessage(input); err != nil {
		input.SetStatus(statConnClosed.Copy(err))
	}
	return input
}

// PreCall temporarily sends TypeCall message and receives message,
// when the session is just builded, do not execute other plugins.
// NOTE:
//  Cannot be called during the Non-PostDial and Non-PostAccept phase;
//  The reply parameter is the body receiver;
//  The external setting seq is invalid, the internal will be forced to set;
//  Does not support automatic redial after disconnection.
func (s *session) PreCall(serviceMethod string, args, reply interface{}, callSetting ...MessageSetting) (opStat *Status) {
	if !s.checkStatus(statusPreparing) {
		return statUnpreparedError
	}
	defer func() {
		if p := recover(); p != nil {
			opStat = statBadMessage.Copy(p, 3)
		}
	}()
	output, opStat := s.send(TypeCall, 0, serviceMethod, args, nil, callSetting)
	if !opStat.OK() {
		socket.PutMessage(output)
		return opStat
	}
	ctx := output.Context()
	socket.PutMessage(output)
	return s.PreReceive(func(Header) interface{} { return reply }, ctx).Status()
}

// PreReply temporarily sends TypeReply message when the session is just builded,
// do not execute other plugins.
// NOTE:
//  Cannot be called during the Non-PostDial and Non-PostAccept phase;
//  The external setting seq is invalid, the internal will be forced to set;
//  Does not support automatic redial after disconnection.
func (s *session) PreReply(req Message, body interface{}, stat *Status, setting ...MessageSetting) (opStat *Status) {
	if !s.checkStatus(statusPreparing) {
		return statUnpreparedError
	}
	var output Message
	defer func() {
		if output != nil {
			socket.PutMessage(output)
		}
		if p := recover(); p != nil {
			opStat = statBadMessage.Copy(p, 3)
		}
	}()
	output, opStat = s.send(TypeReply, req.Seq(), req.ServiceMethod(), body, stat, setting)
	return opStat
}

// RawPush sends a TypePush message without executing other plugins.
// NOTE:
//  The external setting seq is invalid, the internal will be forced to set;
//  Does not support automatic redial after disconnection.
func (s *session) RawPush(serviceMethod string, args interface{}, setting ...MessageSetting) (opStat *Status) {
	var output Message
	defer func() {
		if output != nil {
			socket.PutMessage(output)
		}
		if p := recover(); p != nil {
			opStat = statBadMessage.Copy(p, 3)
		}
	}()
	output, opStat = s.send(TypePush, 0, serviceMethod, args, nil, setting)
	return opStat
}

// Push sends a message of TypePush type, but do not receives reply.
// NOTE:
// If the args is []byte or *[]byte type, it can automatically fill in the body codec name;
// If the session is a client role and PeerConfig.RedialTimes>0, it is automatically re-called once after a failure.
func (s *session) Push(serviceMethod string, args interface{}, setting ...MessageSetting) *Status {
	ctx := s.peer.getContext(s, true)
	defer func() {
		s.peer.putContext(ctx, true)
		if p := recover(); p != nil {
			Errorf("panic:%v\n%s", p, goutil.PanicTrace(2))
		}
	}()

	ctx.start = s.timeNow()
	output := ctx.output
	output.SetMtype(TypePush)
	output.SetServiceMethod(serviceMethod)
	output.SetBody(args)

	for _, fn := range setting {
		if fn != nil {
			fn(output)
		}
	}
	output.SetSeq(atomic.AddInt32(&s.seq, 1))

	if output.BodyCodec() == codec.NilCodecID {
		output.SetBodyCodec(s.peer.defaultBodyCodec)
	}
	if age := s.ContextAge(); age > 0 {
		ctxTimout, _ := context.WithTimeout(output.Context(), age)
		socket.WithContext(ctxTimout)(output)
	}

	stat := s.peer.pluginContainer.preWritePush(ctx)
	if !stat.OK() {
		return stat
	}

	var usedConn net.Conn
W:
	if usedConn, stat = s.write(output); !stat.OK() {
		if stat == statConnClosed && s.redialForClient(usedConn) {
			goto W
		}
		return stat
	}
	if enablePrintRunLog() {
		s.printRunLog("", time.Duration(s.timeNow()-ctx.start), nil, output, typePushLaunch)
	}
	s.peer.pluginContainer.postWritePush(ctx)
	return nil
}

// AsyncCall sends a message and receives reply asynchronously.
// NOTE:
// If the args is []byte or *[]byte type, it can automatically fill in the body codec name;
// If the session is a client role and PeerConfig.RedialTimes>0, it is automatically re-called once after a failure.
func (s *session) AsyncCall(
	serviceMethod string,
	args interface{},
	result interface{},
	callCmdChan chan<- CallCmd,
	setting ...MessageSetting,
) CallCmd {
	if callCmdChan == nil {
		callCmdChan = make(chan CallCmd, 10) // buffered.
	} else {
		// If caller passes callCmdChan != nil, it must arrange that
		// callCmdChan has enough buffer for the number of simultaneous
		// RPCs that will be using that channel. If the channel
		// is totally unbuffered, it's best not to run at all.
		if cap(callCmdChan) == 0 {
			Panicf("*session.AsyncCall(): callCmdChan channel is unbuffered")
		}
	}
	output := socket.NewMessage()
	output.SetServiceMethod(serviceMethod)
	output.SetBody(args)
	output.SetMtype(TypeCall)
	for _, fn := range setting {
		if fn != nil {
			fn(output)
		}
	}

	seq := atomic.AddInt32(&s.seq, 1)
	output.SetSeq(seq)

	if output.BodyCodec() == codec.NilCodecID {
		output.SetBodyCodec(s.peer.defaultBodyCodec)
	}
	if age := s.ContextAge(); age > 0 {
		ctxTimout, _ := context.WithTimeout(output.Context(), age)
		socket.WithContext(ctxTimout)(output)
	}

	cmd := &callCmd{
		sess:        s,
		output:      output,
		result:      result,
		callCmdChan: callCmdChan,
		doneChan:    make(chan struct{}),
		start:       s.timeNow(),
		swap:        goutil.RwMap(),
	}

	// count call-launch
	s.graceCallCmdWaitGroup.Add(1)

	if s.socket.SwapLen() > 0 {
		s.socket.Swap().Range(func(key, value interface{}) bool {
			cmd.swap.Store(key, value)
			return true
		})
	}

	cmd.mu.Lock()
	defer cmd.mu.Unlock()

	s.callCmdMap.Store(seq, cmd)

	defer func() {
		if p := recover(); p != nil {
			Errorf("panic:%v\n%s", p, goutil.PanicTrace(2))
		}
	}()

	cmd.stat = s.peer.pluginContainer.preWriteCall(cmd)
	if !cmd.stat.OK() {
		cmd.done()
		return cmd
	}
	var usedConn net.Conn
W:
	if usedConn, cmd.stat = s.write(output); !cmd.stat.OK() {
		if cmd.stat == statConnClosed && s.redialForClient(usedConn) {
			goto W
		}
		cmd.done()
		return cmd
	}

	s.peer.pluginContainer.postWriteCall(cmd)
	return cmd
}

// Call sends a message and receives reply.
// NOTE:
// If the args is []byte or *[]byte type, it can automatically fill in the body codec name;
// If the session is a client role and PeerConfig.RedialTimes>0, it is automatically re-called once after a failure.
func (s *session) Call(serviceMethod string, args interface{}, result interface{}, setting ...MessageSetting) CallCmd {
	callCmd := s.AsyncCall(serviceMethod, args, result, make(chan CallCmd, 1), setting...)
	<-callCmd.Done()
	return callCmd
}

// Swap returns custom data swap of the session(socket).
func (s *session) Swap() goutil.Map {
	return s.socket.Swap()
}

// Close closes the session.
func (s *session) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.closeLocked()
}

func (s *session) closeLocked() error {
	if !s.tryChangeStatus(statusActiveClosing, statusOk, statusPreparing) {
		return nil
	} // readDisconnected is being called
	s.peer.sessHub.Delete(s.ID())
	s.notifyClosed()
	s.graceCtxWait()
	s.graceCallCmdWaitGroup.Wait()
	s.changeStatus(statusActiveClosed)
	err := s.socket.Close()
	s.peer.pluginContainer.postDisconnect(s)
	return err
}

func (s *session) readDisconnected(oldConn net.Conn, err error) {
	status := s.getStatus()
	switch status {
	case statusPassiveClosed, statusActiveClosed, statusPassiveClosing:
		return
	case statusActiveClosing:
	default:
		s.changeStatus(statusPassiveClosing)
	}

	s.peer.sessHub.Delete(s.ID())

	var reason string
	if err != nil && err != socket.ErrProactivelyCloseSocket {
		if errStr := err.Error(); errStr != "EOF" {
			reason = errStr
			Debugf("disconnect(%s) when reading: %T %s", s.RemoteAddr().String(), err, errStr)
		}
	}
	s.graceCtxWait()

	// cancel the callCmd that is waiting for a reply
	s.callCmdMap.Range(func(_, v interface{}) bool {
		callCmd := v.(*callCmd)
		callCmd.mu.Lock()
		if !callCmd.hasReply() && callCmd.stat.OK() {
			callCmd.cancel(reason)
		}
		callCmd.mu.Unlock()
		return true
	})

	if status == statusActiveClosing {
		return
	}

	s.socket.Close()
	if !s.redialForClient(oldConn) {
		s.changeStatus(statusPassiveClosed)
		s.notifyClosed()
		s.peer.pluginContainer.postDisconnect(s)
	}
}

func (s *session) redialForClient(oldConn net.Conn) bool {
	if s.redialForClientLocked == nil {
		return false
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	// Avoid repeated calls from write and readDisconnected methods
	if oldConn != s.getConn() {
		return true
	}
	if s.tryChangeStatus(statusRedialing, statusOk, statusPassiveClosing, statusPassiveClosed, statusRedialFailed) {
		return s.redialForClientLocked()
	}
	return false
}

func (s *session) startReadAndHandle() {
	var withContext MessageSetting
	if readTimeout := s.SessionAge(); readTimeout > 0 {
		s.socket.SetReadDeadline(coarsetime.CeilingTimeNow().Add(readTimeout))
		ctxTimout, _ := context.WithTimeout(context.Background(), readTimeout)
		withContext = socket.WithContext(ctxTimout)
	} else {
		s.socket.SetReadDeadline(time.Time{})
		withContext = socket.WithContext(nil)
	}

	var (
		err      error
		usedConn = s.getConn()
	)
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic:%v\n%s", p, goutil.PanicTrace(2))
		}
		s.readDisconnected(usedConn, err)
	}()
	// read call, call reply or push
	for s.goonRead() {
		var ctx = s.peer.getContext(s, false)
		withContext(ctx.input)
		if s.peer.pluginContainer.preReadHeader(ctx) != nil {
			s.peer.putContext(ctx, false)
			return
		}
		err = s.socket.ReadMessage(ctx.input)
		if (err != nil && ctx.GetBodyCodec() == codec.NilCodecID) || !s.goonRead() {
			s.peer.putContext(ctx, false)
			return
		}
		if err != nil {
			ctx.stat = statBadMessage.Copy(err)
		}
		s.graceCtxWaitGroup.Add(1)
		if !Go(func() {
			defer s.peer.putContext(ctx, true)
			ctx.handle()
		}) {
			s.peer.putContext(ctx, true)
		}
	}
}

func (s *session) write(message Message) (net.Conn, *Status) {
	usedConn := s.getConn()
	status := s.getStatus()
	if !(status == statusOk || (status == statusActiveClosing && message.Mtype() == TypeReply)) {
		return usedConn, statConnClosed
	}

	var (
		err         error
		ctx         = message.Context()
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
		err = s.socket.WriteMessage(message)
	}

	if err == nil {
		return usedConn, nil
	}

	if err == io.EOF || err == socket.ErrProactivelyCloseSocket {
		return usedConn, statConnClosed
	}

	Debugf("write error: %s", err.Error())

ERR:
	return usedConn, statWriteFailed.Copy(err)
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
	_sess, loaded := sh.sessions.LoadOrStore(sess.ID(), sess)
	if !loaded {
		return
	}
	sh.sessions.Store(sess.ID(), sess)
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
// If second returned arg is false, mean no *session is exist.
func (sh *SessionHub) Random() (*session, bool) {
	_, sess, exist := sh.sessions.Random()
	if !exist {
		return nil, false
	}
	return sess.(*session), true
}

// Len returns the length of the session hub.
// NOTE: the count implemented using sync.Map may be inaccurate.
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
	typeCallLaunch int8 = 3
	typeCallHandle int8 = 4
)

const (
	logFormatPushLaunch = "PUSH-> %s %s %q SEND(%s)"
	logFormatPushHandle = "PUSH<- %s %s %q RECV(%s)"
	logFormatCallLaunch = "CALL-> %s %s %q SEND(%s) RECV(%s)"
	logFormatCallHandle = "CALL<- %s %s %q RECV(%s) SEND(%s)"
)

func enablePrintRunLog() bool {
	return EnableLoggerLevel(WARNING)
}

func (s *session) printRunLog(realIP string, costTime time.Duration, input, output Message, logType int8) {
	var addr = s.RemoteAddr().String()
	if realIP != "" && realIP == addr {
		realIP = "same"
	}
	if realIP == "" {
		realIP = "-"
	}
	addr += "(real:" + realIP + ")"
	var (
		costTimeStr string
		printFunc   = Infof
	)
	if s.peer.countTime {
		if costTime >= s.peer.slowCometDuration {
			costTimeStr = costTime.String() + "(slow)"
			printFunc = Warnf
		} else {
			if GetLoggerLevel() < INFO {
				return
			}
			costTimeStr = costTime.String() + "(fast)"
		}
	} else {
		if GetLoggerLevel() < INFO {
			return
		}
		costTimeStr = "(-)"
	}

	switch logType {
	case typePushLaunch:
		printFunc(logFormatPushLaunch, addr, costTimeStr, output.ServiceMethod(), messageLogBytes(output, s.peer.printDetail))
	case typePushHandle:
		printFunc(logFormatPushHandle, addr, costTimeStr, input.ServiceMethod(), messageLogBytes(input, s.peer.printDetail))
	case typeCallLaunch:
		printFunc(logFormatCallLaunch, addr, costTimeStr, output.ServiceMethod(), messageLogBytes(output, s.peer.printDetail), messageLogBytes(input, s.peer.printDetail))
	case typeCallHandle:
		printFunc(logFormatCallHandle, addr, costTimeStr, input.ServiceMethod(), messageLogBytes(input, s.peer.printDetail), messageLogBytes(output, s.peer.printDetail))
	}
}

func messageLogBytes(message Message, printDetail bool) []byte {
	var b = make([]byte, 0, 128)
	b = append(b, '{')
	b = append(b, '"', 's', 'i', 'z', 'e', '"', ':')
	b = append(b, strconv.FormatUint(uint64(message.Size()), 10)...)
	if statBytes := message.Status().EncodeQuery(); len(statBytes) > 0 {
		b = append(b, ',', '"', 's', 't', 'a', 't', 'u', 's', '"', ':')
		b = append(b, statBytes...)
	}
	if printDetail {
		if message.Meta().Len() > 0 {
			b = append(b, ',', '"', 'm', 'e', 't', 'a', '"', ':')
			b = append(b, utils.ToJSONStr(message.Meta().QueryString(), false)...)
		}
		if bodyBytes := bodyLogBytes(message); len(bodyBytes) > 0 {
			b = append(b, ',', '"', 'b', 'o', 'd', 'y', '"', ':')
			b = append(b, bodyBytes...)
		}
	}
	b = append(b, '}')
	return b
}

func bodyLogBytes(message Message) []byte {
	switch v := message.Body().(type) {
	case nil:
		return nil
	case []byte:
		return utils.ToJSONStr(v, false)
	case *[]byte:
		return utils.ToJSONStr(*v, false)
	}
	b, _ := json.Marshal(message.Body())
	return b
}
