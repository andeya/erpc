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
	"reflect"
	"sync"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/socket"
	"github.com/henrylee2cn/teleport/utils"
)

type (
	// PreCtx context method set used before reading message header.
	PreCtx interface {
		// Peer returns the peer.
		Peer() Peer
		// Session returns the session.
		Session() CtxSession
		// IP returns the remote addr.
		IP() string
		// RealIP returns the the current real remote addr.
		RealIP() string
		// Swap returns custom data swap of context.
		Swap() goutil.Map
		// Context carries a deadline, a cancelation signal, and other values across
		// API boundaries.
		Context() context.Context
		// Logger logger interface
		Logger
	}
	// WriteCtx context method set for writing message.
	WriteCtx interface {
		PreCtx
		// Output returns writed message.
		Output() Message
		// StatusOK returns the handle status is OK or not.
		StatusOK() bool
		// Status returns the handle status.
		Status() *Status
	}
	// inputCtx common context method set.
	inputCtx interface {
		PreCtx
		// Seq returns the input message sequence.
		Seq() int32
		// PeekMeta peeks the header metadata for the input message.
		PeekMeta(key string) []byte
		// VisitMeta calls f for each existing metadata.
		//
		// f must not retain references to key and value after returning.
		// Make key and/or value copies if you need storing them after returning.
		VisitMeta(f func(key, value []byte))
		// CopyMeta returns the input message metadata copy.
		CopyMeta() *utils.Args
		// ServiceMethod returns the input message service method.
		ServiceMethod() string
		// ResetServiceMethod resets the input message service method.
		ResetServiceMethod(string)
	}
	// ReadCtx context method set for reading message.
	ReadCtx interface {
		inputCtx
		// Input returns readed message.
		Input() Message
		// StatusOK returns the handle status is OK or not.
		StatusOK() bool
		// Status returns the handle status.
		Status() *Status
	}
	// PushCtx context method set for handling the pushed message.
	// For example:
	//  type HomePush struct{ PushCtx }
	PushCtx interface {
		inputCtx
		// GetBodyCodec gets the body codec type of the input message.
		GetBodyCodec() byte
	}
	// CallCtx context method set for handling the called message.
	// For example:
	//  type HomeCall struct{ CallCtx }
	CallCtx interface {
		inputCtx
		// Input returns readed message.
		Input() Message
		// GetBodyCodec gets the body codec type of the input message.
		GetBodyCodec() byte
		// Output returns writed message.
		Output() Message
		// ReplyBodyCodec initializes and returns the reply message body codec id.
		ReplyBodyCodec() byte
		// SetBodyCodec sets the body codec for reply message.
		SetBodyCodec(byte)
		// AddMeta adds the header metadata 'key=value' for reply message.
		// Multiple values for the same key may be added.
		AddMeta(key, value string)
		// SetMeta sets the header metadata 'key=value' for reply message.
		SetMeta(key, value string)
		// AddXferPipe appends transfer filter pipe of reply message.
		AddXferPipe(filterID ...byte)
	}
	// UnknownPushCtx context method set for handling the unknown pushed message.
	UnknownPushCtx interface {
		inputCtx
		// GetBodyCodec gets the body codec type of the input message.
		GetBodyCodec() byte
		// InputBodyBytes if the input body binder is []byte type, returns it, else returns nil.
		InputBodyBytes() []byte
		// Bind when the raw body binder is []byte type, now binds the input body to v.
		Bind(v interface{}) (bodyCodec byte, err error)
	}
	// UnknownCallCtx context method set for handling the unknown called message.
	UnknownCallCtx interface {
		inputCtx
		// GetBodyCodec gets the body codec type of the input message.
		GetBodyCodec() byte
		// InputBodyBytes if the input body binder is []byte type, returns it, else returns nil.
		InputBodyBytes() []byte
		// Bind when the raw body binder is []byte type, now binds the input body to v.
		Bind(v interface{}) (bodyCodec byte, err error)
		// SetBodyCodec sets the body codec for reply message.
		SetBodyCodec(byte)
		// AddMeta adds the header metadata 'key=value' for reply message.
		// Multiple values for the same key may be added.
		AddMeta(key, value string)
		// SetMeta sets the header metadata 'key=value' for reply message.
		SetMeta(key, value string)
		// AddXferPipe appends transfer filter pipe of reply message.
		AddXferPipe(filterID ...byte)
	}
)

var (
	_ Logger         = new(handlerCtx)
	_ PreCtx         = new(handlerCtx)
	_ inputCtx       = new(handlerCtx)
	_ WriteCtx       = new(handlerCtx)
	_ ReadCtx        = new(handlerCtx)
	_ PushCtx        = new(handlerCtx)
	_ CallCtx        = new(handlerCtx)
	_ UnknownPushCtx = new(handlerCtx)
	_ UnknownCallCtx = new(handlerCtx)
)

// handlerCtx the underlying common instance of CallCtx and PushCtx.
type handlerCtx struct {
	sess            *session
	input           Message
	output          Message
	handler         *Handler
	arg             reflect.Value
	callCmd         *callCmd
	swap            goutil.Map
	start           int64
	cost            time.Duration
	pluginContainer *PluginContainer
	stat            *Status
	context         context.Context
}

var (
	emptyValue  = reflect.Value{}
	emptyMethod = reflect.Method{}
)

// newReadHandleCtx creates a handlerCtx for one request/response or push.
func newReadHandleCtx() *handlerCtx {
	c := new(handlerCtx)
	c.input = socket.NewMessage()
	c.input.SetNewBody(c.binding)
	c.output = socket.NewMessage()
	return c
}

func (c *handlerCtx) reInit(s *session) {
	c.sess = s
	count := s.socket.SwapLen()
	c.swap = goutil.RwMap(count)
	if count > 0 {
		s.socket.Swap().Range(func(key, value interface{}) bool {
			c.swap.Store(key, value)
			return true
		})
	}
}

func (c *handlerCtx) clean() {
	c.sess = nil
	c.handler = nil
	c.arg = emptyValue
	c.callCmd = nil
	c.swap = nil
	c.cost = 0
	c.pluginContainer = nil
	c.stat = nil
	c.context = nil
	c.input.Reset(socket.WithNewBody(c.binding))
	c.output.Reset()
}

// Peer returns the peer.
func (c *handlerCtx) Peer() Peer {
	return c.sess.peer
}

// Session returns the session.
func (c *handlerCtx) Session() CtxSession {
	return c.sess
}

// Input returns readed message.
func (c *handlerCtx) Input() Message {
	return c.input
}

// Output returns writed message.
func (c *handlerCtx) Output() Message {
	return c.output
}

// Swap returns custom data swap of context.
func (c *handlerCtx) Swap() goutil.Map {
	return c.swap
}

// Seq returns the input message sequence.
func (c *handlerCtx) Seq() int32 {
	return c.input.Seq()
}

// ServiceMethod returns the input message service method.
func (c *handlerCtx) ServiceMethod() string {
	return c.input.ServiceMethod()
}

// ResetServiceMethod resets the input message service method.
func (c *handlerCtx) ResetServiceMethod(serviceMethod string) {
	c.input.SetServiceMethod(serviceMethod)
}

// PeekMeta peeks the header metadata for the input message.
func (c *handlerCtx) PeekMeta(key string) []byte {
	return c.input.Meta().Peek(key)
}

// VisitMeta calls f for each existing metadata.
//
// f must not retain references to key and value after returning.
// Make key and/or value copies if you need storing them after returning.
func (c *handlerCtx) VisitMeta(f func(key, value []byte)) {
	c.input.Meta().VisitAll(f)
}

// CopyMeta returns the input message metadata copy.
func (c *handlerCtx) CopyMeta() *utils.Args {
	dst := utils.AcquireArgs()
	c.input.Meta().CopyTo(dst)
	return dst
}

// AddMeta adds the header metadata 'key=value' for reply message.
// Multiple values for the same key may be added.
func (c *handlerCtx) AddMeta(key, value string) {
	c.output.Meta().Add(key, value)
}

// SetMeta sets the header metadata 'key=value' for reply message.
func (c *handlerCtx) SetMeta(key, value string) {
	c.output.Meta().Set(key, value)
}

// GetBodyCodec gets the body codec type of the input message.
func (c *handlerCtx) GetBodyCodec() byte {
	return c.input.BodyCodec()
}

// SetBodyCodec sets the body codec for reply message.
func (c *handlerCtx) SetBodyCodec(bodyCodec byte) {
	c.output.SetBodyCodec(bodyCodec)
}

// AddXferPipe appends transfer filter pipe of reply message.
func (c *handlerCtx) AddXferPipe(filterID ...byte) {
	c.output.XferPipe().Append(filterID...)
}

// IP returns the remote addr.
func (c *handlerCtx) IP() string {
	return c.sess.RemoteAddr().String()
}

// RealIP returns the the current real remote addr.
func (c *handlerCtx) RealIP() string {
	realIP := c.PeekMeta(MetaRealIP)
	if len(realIP) > 0 {
		return string(realIP)
	}
	return c.sess.RemoteAddr().String()
}

// Context carries a deadline, a cancelation signal, and other values across
// API boundaries.
func (c *handlerCtx) Context() context.Context {
	if c.context == nil {
		return c.input.Context()
	}
	return c.context
}

// setContext sets the context for timeout.
func (c *handlerCtx) setContext(ctx context.Context) {
	c.context = ctx
}

// Be executed synchronously when reading message
func (c *handlerCtx) binding(header Header) (body interface{}) {
	c.start = c.sess.timeNow()
	c.pluginContainer = c.sess.peer.pluginContainer
	switch header.Mtype() {
	case TypeReply:
		return c.bindReply(header)
	case TypePush:
		return c.bindPush(header)
	case TypeCall:
		return c.bindCall(header)
	default:
		c.stat = statCodeMtypeNotAllowed
		return nil
	}
}

const logFormatDisconnected = "disconnected due to unsupported message type: %d %s %s %q RECV(%s)"

// Be executed asynchronously after readed message
func (c *handlerCtx) handle() {
	if c.stat.Code() == CodeMtypeNotAllowed {
		goto E
	}
	switch c.input.Mtype() {
	case TypeReply:
		// handles call reply
		c.handleReply()
		return

	case TypePush:
		//  handles push
		c.handlePush()
		return

	case TypeCall:
		// handles and replies call
		c.handleCall()
		return

	default:
	}
E:
	// if unsupported, disconnected.
	c.output.SetStatus(statCodeMtypeNotAllowed)
	Errorf(logFormatDisconnected,
		c.input.Mtype(), c.IP(), c.input.ServiceMethod(), c.input.Seq(),
		messageLogBytes(c.input, c.sess.peer.printDetail))
	go c.sess.Close()
}

func (c *handlerCtx) bindPush(header Header) interface{} {
	c.stat = c.pluginContainer.postReadPushHeader(c)
	if !c.stat.OK() {
		return nil
	}

	if len(header.ServiceMethod()) == 0 {
		c.stat = statBadMessage.Copy("invalid service method for message")
		return nil
	}

	var ok bool
	c.handler, ok = c.sess.getPushHandler(header.ServiceMethod())
	if !ok {
		c.stat = statNotFound
		return nil
	}

	// reset plugin container
	c.pluginContainer = c.handler.pluginContainer

	c.arg = c.handler.NewArgValue()
	c.input.SetBody(c.arg.Interface())
	c.stat = c.pluginContainer.preReadPushBody(c)
	if !c.stat.OK() {
		return nil
	}

	return c.input.Body()
}

func (c *handlerCtx) recordCost() {
	c.cost = time.Duration(c.sess.timeNow() - c.start)
}

// handlePush handles push.
func (c *handlerCtx) handlePush() {
	if age := c.sess.ContextAge(); age > 0 {
		ctxTimout, _ := context.WithTimeout(context.Background(), age)
		c.setContext(ctxTimout)
	}
	defer func() {
		if p := recover(); p != nil {
			Errorf("panic:%v\n%s", p, goutil.PanicTrace(2))
		}
		c.recordCost()
		if enablePrintRunLog() {
			c.sess.printRunLog(c.RealIP(), c.cost, c.input, nil, typePushHandle)
		}
	}()
	if c.stat.OK() && c.handler != nil {
		if c.pluginContainer.postReadPushBody(c) == nil {
			if c.handler.isUnknown {
				c.handler.unknownHandleFunc(c)
			} else {
				c.handler.handleFunc(c, c.arg)
			}
		}
	}
	if !c.stat.OK() {
		Warnf("%s", c.stat.String())
	}
}

func (c *handlerCtx) bindCall(header Header) interface{} {
	c.stat = c.pluginContainer.postReadCallHeader(c)
	if !c.stat.OK() {
		return nil
	}

	if len(header.ServiceMethod()) == 0 {
		c.stat = statBadMessage.Copy("invalid service method for message")
		return nil
	}

	var ok bool
	c.handler, ok = c.sess.getCallHandler(header.ServiceMethod())
	if !ok {
		c.stat = statNotFound
		return nil
	}

	// reset plugin container
	c.pluginContainer = c.handler.pluginContainer

	if c.handler.isUnknown {
		c.input.SetBody(new([]byte))
	} else {
		c.arg = c.handler.NewArgValue()
		c.input.SetBody(c.arg.Interface())
	}

	c.stat = c.pluginContainer.preReadCallBody(c)
	if !c.stat.OK() {
		return nil
	}

	return c.input.Body()
}

// handleCall handles and replies call.
func (c *handlerCtx) handleCall() {
	var writed bool
	defer func() {
		if p := recover(); p != nil {
			Errorf("panic:%v\n%s", p, goutil.PanicTrace(2))
			if !writed {
				if c.stat.OK() {
					c.stat = statInternalServerError.Copy(p)
				}
				c.writeReply(c.stat)
			}
		}
		c.recordCost()
		if enablePrintRunLog() {
			c.sess.printRunLog(c.RealIP(), c.cost, c.input, c.output, typeCallHandle)
		}
	}()

	c.output.SetMtype(TypeReply)
	c.output.SetSeq(c.input.Seq())
	c.output.SetServiceMethod(c.input.ServiceMethod())
	c.output.XferPipe().AppendFrom(c.input.XferPipe())

	if age := c.sess.ContextAge(); age > 0 {
		ctxTimout, _ := context.WithTimeout(c.input.Context(), age)
		c.setContext(ctxTimout)
		socket.WithContext(ctxTimout)(c.output)
	}

	if c.stat.OK() {
		c.stat = c.output.Status()
	}

	// handle call
	if c.stat.OK() {
		c.stat = c.pluginContainer.postReadCallBody(c)
		if c.stat.OK() {
			if c.handler.isUnknown {
				c.handler.unknownHandleFunc(c)
			} else {
				c.handler.handleFunc(c, c.arg)
			}
		}
	}

	// reply call
	c.setReplyBodyCodec(!c.stat.OK())
	c.pluginContainer.preWriteReply(c)
	stat := c.writeReply(c.stat)
	if !stat.OK() {
		if c.stat.OK() {
			c.stat = stat
		}
		if stat.Code() != CodeConnClosed {
			c.writeReply(statInternalServerError.Copy(stat.Cause()))
		}
		return
	}
	writed = true
	c.pluginContainer.postWriteReply(c)
}

// ReplyBodyCodec initializes and returns the reply message body codec id.
func (c *handlerCtx) ReplyBodyCodec() byte {
	id := c.output.BodyCodec()
	if id != codec.NilCodecID {
		return id
	}
	id, ok := GetAcceptBodyCodec(c.input.Meta())
	if ok {
		if _, err := codec.Get(id); err == nil {
			c.output.SetBodyCodec(id)
			return id
		}
	}
	id = c.input.BodyCodec()
	c.output.SetBodyCodec(id)
	return id
}

func (c *handlerCtx) setReplyBodyCodec(hasError bool) {
	if hasError {
		return
	}
	c.ReplyBodyCodec()
}

func (c *handlerCtx) writeReply(stat *Status) *Status {
	if !stat.OK() {
		c.output.SetStatus(stat)
		c.output.SetBody(nil)
		c.output.SetBodyCodec(codec.NilCodecID)
	}
	serviceMethod := c.output.ServiceMethod()
	c.output.SetServiceMethod("")
	_, stat = c.sess.write(c.output)
	c.output.SetServiceMethod(serviceMethod)
	return stat
}

func (c *handlerCtx) bindReply(header Header) interface{} {
	_callCmd, ok := c.sess.callCmdMap.Load(header.Seq())
	if !ok {
		Warnf("not found call cmd: %v", c.input)
		return nil
	}
	c.callCmd = _callCmd.(*callCmd)

	// unlock: handleReply
	c.callCmd.mu.Lock()
	c.input.SetServiceMethod(c.callCmd.output.ServiceMethod())
	c.swap = c.callCmd.swap
	c.callCmd.inputBodyCodec = c.GetBodyCodec()
	// if c.callCmd.inputMeta!=nil, means the callCmd is replyed.
	c.callCmd.inputMeta = utils.AcquireArgs()
	c.input.Meta().CopyTo(c.callCmd.inputMeta)
	c.setContext(c.callCmd.output.Context())
	c.input.SetBody(c.callCmd.result)

	stat := c.pluginContainer.postReadReplyHeader(c)
	if !stat.OK() {
		c.callCmd.stat = stat
		return nil
	}
	stat = c.pluginContainer.preReadReplyBody(c)
	if !stat.OK() {
		c.callCmd.stat = stat
		return nil
	}
	return c.input.Body()
}

// handleReply handles call reply.
func (c *handlerCtx) handleReply() {
	if c.callCmd == nil {
		return
	}
	defer func() {
		if p := recover(); p != nil {
			Errorf("panic:%v\n%s", p, goutil.PanicTrace(2))
		}
		c.callCmd.result = c.input.Body()
		c.stat = c.callCmd.stat
		c.callCmd.done()
		c.callCmd.cost = time.Duration(c.sess.timeNow() - c.callCmd.start)
		if enablePrintRunLog() {
			c.sess.printRunLog(c.RealIP(), c.callCmd.cost, c.input, c.callCmd.output, typeCallLaunch)
		}
		// lock: bindReply
		c.callCmd.mu.Unlock()
	}()
	if c.callCmd.stat.OK() {
		stat := c.input.Status()
		if stat.OK() {
			stat = c.pluginContainer.postReadReplyBody(c)
		}
		c.callCmd.stat = stat
	}
}

// StatusOK returns the handle status is OK or not.
func (c *handlerCtx) StatusOK() bool {
	return c.stat.OK()
}

// Status returns the handle status.
func (c *handlerCtx) Status() *Status {
	return c.stat
}

// InputBodyBytes if the input body binder is []byte type, returns it, else returns nil.
func (c *handlerCtx) InputBodyBytes() []byte {
	b, ok := c.input.Body().(*[]byte)
	if !ok {
		return nil
	}
	return *b
}

// Bind when the raw body binder is []byte type, now binds the input body to v.
func (c *handlerCtx) Bind(v interface{}) (byte, error) {
	b := c.InputBodyBytes()
	if b == nil {
		return codec.NilCodecID, nil
	}
	c.input.SetBody(v)
	err := c.input.UnmarshalBody(b)
	return c.input.BodyCodec(), err
}

type (
	// CallCmd the command of the calling operation's response.
	CallCmd interface {
		// TracePeer trace back the peer.
		TracePeer() (peer Peer, found bool)
		// TraceSession trace back the session.
		TraceSession() (sess Session, found bool)
		// Context carries a deadline, a cancelation signal, and other values across
		// API boundaries.
		Context() context.Context
		// Output returns writed message.
		Output() Message
		// StatusOK returns the call status is OK or not.
		StatusOK() bool
		// Status returns the call status.
		Status() *Status
		// Done returns the chan that indicates whether it has been completed.
		Done() <-chan struct{}
		// Reply returns the call reply.
		// NOTE:
		//  Inside, <-Done() is automatically called and blocked,
		//  until the call is completed!
		Reply() (interface{}, *Status)
		// InputBodyCodec gets the body codec type of the input message.
		// NOTE:
		//  Inside, <-Done() is automatically called and blocked,
		//  until the call is completed!
		InputBodyCodec() byte
		// InputMeta returns the header metadata of input message.
		// NOTE:
		//  Inside, <-Done() is automatically called and blocked,
		//  until the call is completed!
		InputMeta() *utils.Args
		// CostTime returns the called cost time.
		// If PeerConfig.CountTime=false, always returns 0.
		// NOTE:
		//  Inside, <-Done() is automatically called and blocked,
		//  until the call is completed!
		CostTime() time.Duration
	}
	callCmd struct {
		start          int64
		cost           time.Duration
		sess           *session
		output         Message
		result         interface{}
		stat           *Status
		inputMeta      *utils.Args
		swap           goutil.Map
		mu             sync.Mutex
		callCmdChan    chan<- CallCmd // Send itself to the public channel when call is complete.
		doneChan       chan struct{}  // Strobes when call is complete.
		inputBodyCodec byte
	}
)

var _ WriteCtx = new(callCmd)

// TracePeer trace back the peer.
func (c *callCmd) TracePeer() (Peer, bool) {
	return c.Peer(), true
}

// Peer returns the peer.
func (c *callCmd) Peer() Peer {
	return c.sess.peer
}

// TraceSession trace back the session.
func (c *callCmd) TraceSession() (Session, bool) {
	return c.sess, true
}

// Session returns the session.
func (c *callCmd) Session() CtxSession {
	return c.sess
}

// IP returns the remote addr.
func (c *callCmd) IP() string {
	return c.sess.RemoteAddr().String()
}

// RealIP returns the the current real remote addr.
func (c *callCmd) RealIP() string {
	realIP := c.inputMeta.Peek(MetaRealIP)
	if len(realIP) > 0 {
		return string(realIP)
	}
	return c.sess.RemoteAddr().String()
}

// Swap returns custom data swap of context.
func (c *callCmd) Swap() goutil.Map {
	return c.swap
}

// SwapLen returns the amount of recorded custom data of context.
func (c *callCmd) SwapLen() int {
	return c.swap.Len()
}

// Output returns writed message.
func (c *callCmd) Output() Message {
	return c.output
}

// Context carries a deadline, a cancelation signal, and other values across
// API boundaries.
func (c *callCmd) Context() context.Context {
	return c.output.Context()
}

// StatusOK returns the call status is OK or not.
func (c *callCmd) StatusOK() bool {
	return c.stat.OK()
}

// Status returns the call status.
func (c *callCmd) Status() *Status {
	return c.stat
}

// Done returns the chan that indicates whether it has been completed.
func (c *callCmd) Done() <-chan struct{} {
	return c.doneChan
}

// Reply returns the call reply.
// NOTE:
//  Inside, <-Done() is automatically called and blocked,
//  until the call is completed!
func (c *callCmd) Reply() (interface{}, *Status) {
	<-c.Done()
	return c.result, c.stat
}

// InputBodyCodec gets the body codec type of the input message.
// NOTE:
//  Inside, <-Done() is automatically called and blocked,
//  until the call is completed!
func (c *callCmd) InputBodyCodec() byte {
	<-c.Done()
	return c.inputBodyCodec
}

// InputMeta returns the header metadata of input message.
// NOTE:
//  Inside, <-Done() is automatically called and blocked,
//  until the call is completed!
func (c *callCmd) InputMeta() *utils.Args {
	<-c.Done()
	return c.inputMeta
}

// CostTime returns the called cost time.
// If PeerConfig.CountTime=false, always returns 0.
// NOTE:
//  Inside, <-Done() is automatically called and blocked,
//  until the call is completed!
func (c *callCmd) CostTime() time.Duration {
	<-c.Done()
	return c.cost
}

func (c *callCmd) done() {
	c.sess.callCmdMap.Delete(c.output.Seq())
	c.callCmdChan <- c
	close(c.doneChan)
	// free count call-launch
	c.sess.graceCallCmdWaitGroup.Done()
}

func (c *callCmd) cancel(reason string) {
	c.sess.callCmdMap.Delete(c.output.Seq())
	if reason != "" {
		c.stat = statConnClosed.Copy(reason)
	} else {
		c.stat = statConnClosed
	}
	c.callCmdChan <- c
	close(c.doneChan)
	// free count call-launch
	c.sess.graceCallCmdWaitGroup.Done()
}

// if callCmd.inputMeta!=nil, means the callCmd is replyed.
func (c *callCmd) hasReply() bool {
	return c.inputMeta != nil
}
