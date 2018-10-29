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
	"fmt"
	"net/url"
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
		Session() Session
		// Ip returns the remote addr.
		Ip() string
		// RealIp returns the the current real remote addr.
		RealIp() string
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
		Output() *Message
		// Rerror returns the handle error.
		Rerror() *Rerror
	}
	// inputCtx common context method set.
	inputCtx interface {
		PreCtx
		// Seq returns the input message sequence.
		Seq() string
		// PeekMeta peeks the header metadata for the input message.
		PeekMeta(key string) []byte
		// VisitMeta calls f for each existing metadata.
		//
		// f must not retain references to key and value after returning.
		// Make key and/or value copies if you need storing them after returning.
		VisitMeta(f func(key, value []byte))
		// CopyMeta returns the input message metadata copy.
		CopyMeta() *utils.Args
		// Uri returns the input message uri.
		Uri() string
		// UriObject returns the input message uri object.
		UriObject() *url.URL
		// ResetUri resets the input message uri.
		ResetUri(string)
		// Path returns the input message uri path.
		Path() string
		// Query returns the input message uri query object.
		Query() url.Values
	}
	// ReadCtx context method set for reading message.
	ReadCtx interface {
		inputCtx
		// Input returns readed message.
		Input() *Message
		// Rerror returns the handle error.
		Rerror() *Rerror
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
		Input() *Message
		// GetBodyCodec gets the body codec type of the input message.
		GetBodyCodec() byte
		// Output returns writed message.
		Output() *Message
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
		AddXferPipe(filterId ...byte)
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
		AddXferPipe(filterId ...byte)
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
	input           *Message
	output          *Message
	handler         *Handler
	arg             reflect.Value
	callCmd         *callCmd
	swap            goutil.Map
	start           time.Time
	cost            time.Duration
	pluginContainer *PluginContainer
	handleErr       *Rerror
	context         context.Context
	next            *handlerCtx
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
	c.handleErr = nil
	c.context = nil
	c.input.Reset(socket.WithNewBody(c.binding))
	c.output.Reset()
}

// Peer returns the peer.
func (c *handlerCtx) Peer() Peer {
	return c.sess.peer
}

// Session returns the session.
func (c *handlerCtx) Session() Session {
	return c.sess
}

// Input returns readed message.
func (c *handlerCtx) Input() *Message {
	return c.input
}

// Output returns writed message.
func (c *handlerCtx) Output() *Message {
	return c.output
}

// Swap returns custom data swap of context.
func (c *handlerCtx) Swap() goutil.Map {
	return c.swap
}

// Seq returns the input message sequence.
func (c *handlerCtx) Seq() string {
	return c.input.Seq()
}

// Uri returns the input message uri string.
func (c *handlerCtx) Uri() string {
	return c.input.Uri()
}

// UriObject returns the input message uri object.
func (c *handlerCtx) UriObject() *url.URL {
	return c.input.UriObject()
}

// ResetUri resets the input message uri.
func (c *handlerCtx) ResetUri(uri string) {
	c.input.SetUri(uri)
}

// Path returns the input message uri path.
func (c *handlerCtx) Path() string {
	return c.input.UriObject().Path
}

// Query returns the input message uri query object.
func (c *handlerCtx) Query() url.Values {
	return c.input.UriObject().Query()
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
func (c *handlerCtx) AddXferPipe(filterId ...byte) {
	c.output.XferPipe().Append(filterId...)
}

// Ip returns the remote addr.
func (c *handlerCtx) Ip() string {
	return c.sess.RemoteAddr().String()
}

// RealIp returns the the current real remote addr.
func (c *handlerCtx) RealIp() string {
	realIp := c.PeekMeta(MetaRealIp)
	if len(realIp) > 0 {
		return string(realIp)
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
		c.handleErr = rerrCodeMtypeNotAllowed
		return nil
	}
}

const logFormatDisconnected = "disconnected due to unsupported message type: %d\n%s %s %q\nRECV(%s)"

// Be executed asynchronously after readed message
func (c *handlerCtx) handle() {
	if c.handleErr != nil && c.handleErr.Code == CodeMtypeNotAllowed {
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
	rerrCodeMtypeNotAllowed.SetToMeta(c.output.Meta())
	Errorf(logFormatDisconnected, c.input.Mtype(), c.Ip(), c.input.Uri(), c.input.Seq(), messageLogBytes(c.input, c.sess.peer.printDetail))
	go c.sess.Close()
}

func (c *handlerCtx) bindPush(header Header) interface{} {
	c.handleErr = c.pluginContainer.postReadPushHeader(c)
	if c.handleErr != nil {
		return nil
	}

	u := header.UriObject()
	if len(u.Path) == 0 {
		c.handleErr = rerrBadMessage.Copy().SetReason("invalid URI for message")
		return nil
	}

	var ok bool
	c.handler, ok = c.sess.getPushHandler(u.Path)
	if !ok {
		c.handleErr = rerrNotFound
		return nil
	}

	// reset plugin container
	c.pluginContainer = c.handler.pluginContainer

	c.arg = c.handler.NewArgValue()
	c.input.SetBody(c.arg.Interface())
	c.handleErr = c.pluginContainer.preReadPushBody(c)
	if c.handleErr != nil {
		return nil
	}

	return c.input.Body()
}

// handlePush handles push.
func (c *handlerCtx) handlePush() {
	if age := c.sess.ContextAge(); age > 0 {
		ctxTimout, _ := context.WithTimeout(context.Background(), age)
		c.setContext(ctxTimout)
	}

	defer func() {
		if p := recover(); p != nil {
			Debugf("panic:%v\n%s", p, goutil.PanicTrace(2))
		}
		c.cost = c.sess.timeSince(c.start)
		c.sess.printAccessLog(c.RealIp(), c.cost, c.input, nil, typePushHandle)
	}()

	if c.handleErr == nil && c.handler != nil {
		if c.pluginContainer.postReadPushBody(c) == nil {
			if c.handler.isUnknown {
				c.handler.unknownHandleFunc(c)
			} else {
				c.handler.handleFunc(c, c.arg)
			}
		}
	}
	if c.handleErr != nil {
		Warnf("%s", c.handleErr.String())
	}
}

func (c *handlerCtx) bindCall(header Header) interface{} {
	c.handleErr = c.pluginContainer.postReadCallHeader(c)
	if c.handleErr != nil {
		return nil
	}

	u := header.UriObject()
	if len(u.Path) == 0 {
		c.handleErr = rerrBadMessage.Copy().SetReason("invalid URI for message")
		return nil
	}

	var ok bool
	c.handler, ok = c.sess.getCallHandler(u.Path)
	if !ok {
		c.handleErr = rerrNotFound
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

	c.handleErr = c.pluginContainer.preReadCallBody(c)
	if c.handleErr != nil {
		return nil
	}

	return c.input.Body()
}

// handleCall handles and replies call.
func (c *handlerCtx) handleCall() {
	var writed bool
	defer func() {
		if p := recover(); p != nil {
			Debugf("panic:%v\n%s", p, goutil.PanicTrace(2))
			if !writed {
				if c.handleErr == nil {
					c.handleErr = rerrInternalServerError.Copy().SetReason(fmt.Sprint(p))
				}
				c.writeReply(c.handleErr)
			}
		}
		c.cost = c.sess.timeSince(c.start)
		c.sess.printAccessLog(c.RealIp(), c.cost, c.input, c.output, typeCallHandle)
	}()

	c.output.SetMtype(TypeReply)
	c.output.SetSeq(c.input.Seq())
	c.output.SetUriObject(c.input.UriObject())
	c.output.XferPipe().AppendFrom(c.input.XferPipe())

	if age := c.sess.ContextAge(); age > 0 {
		ctxTimout, _ := context.WithTimeout(c.input.Context(), age)
		c.setContext(ctxTimout)
		socket.WithContext(ctxTimout)(c.output)
	}

	if c.handleErr == nil {
		c.handleErr = NewRerrorFromMeta(c.output.Meta())
	}

	// handle call
	if c.handleErr == nil {
		c.handleErr = c.pluginContainer.postReadCallBody(c)
		if c.handleErr == nil {
			if c.handler.isUnknown {
				c.handler.unknownHandleFunc(c)
			} else {
				c.handler.handleFunc(c, c.arg)
			}
		}
	}

	// reply call
	c.setReplyBodyCodec(c.handleErr != nil)
	c.pluginContainer.preWriteReply(c)
	rerr := c.writeReply(c.handleErr)
	if rerr != nil {
		if c.handleErr == nil {
			c.handleErr = rerr
		}
		if rerr != rerrConnClosed {
			c.writeReply(rerrInternalServerError.Copy().SetReason(rerr.Reason))
		}
		return
	}
	writed = true
	c.pluginContainer.postWriteReply(c)
}

// ReplyBodyCodec initializes and returns the reply message body codec id.
func (c *handlerCtx) ReplyBodyCodec() byte {
	id := c.output.BodyCodec()
	if id != codec.NilCodecId {
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

func (c *handlerCtx) writeReply(rerr *Rerror) *Rerror {
	if rerr != nil {
		rerr.SetToMeta(c.output.Meta())
		c.output.SetBody(nil)
		c.output.SetBodyCodec(codec.NilCodecId)
	}
	uri := c.output.Uri()
	c.output.SetUri("")
	_, rerr = c.sess.write(c.output)
	c.output.SetUri(uri)
	return rerr
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
	c.input.SetUri(c.callCmd.output.Uri())
	c.swap = c.callCmd.swap
	c.callCmd.inputBodyCodec = c.GetBodyCodec()
	// if c.callCmd.inputMeta!=nil, means the callCmd is replyed.
	c.callCmd.inputMeta = utils.AcquireArgs()
	c.input.Meta().CopyTo(c.callCmd.inputMeta)
	c.setContext(c.callCmd.output.Context())
	c.input.SetBody(c.callCmd.result)

	rerr := c.pluginContainer.postReadReplyHeader(c)
	if rerr != nil {
		c.callCmd.rerr = rerr
		return nil
	}
	rerr = c.pluginContainer.preReadReplyBody(c)
	if rerr != nil {
		c.callCmd.rerr = rerr
		return nil
	}
	return c.input.Body()
}

// handleReply handles call reply.
func (c *handlerCtx) handleReply() {
	if c.callCmd == nil {
		return
	}

	// lock: bindReply
	defer c.callCmd.mu.Unlock()

	defer func() {
		if p := recover(); p != nil {
			Debugf("panic:%v\n%s", p, goutil.PanicTrace(2))
		}
		c.callCmd.result = c.input.Body()
		c.handleErr = c.callCmd.rerr
		c.callCmd.done()
		c.callCmd.cost = c.sess.timeSince(c.callCmd.start)
		c.sess.printAccessLog(c.RealIp(), c.callCmd.cost, c.input, c.callCmd.output, typeCallLaunch)
	}()
	if c.callCmd.rerr != nil {
		return
	}
	rerr := NewRerrorFromMeta(c.input.Meta())
	if rerr == nil {
		rerr = c.pluginContainer.postReadReplyBody(c)
	}
	c.callCmd.rerr = rerr
}

// Rerror returns the handle error.
func (c *handlerCtx) Rerror() *Rerror {
	return c.handleErr
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
		return codec.NilCodecId, nil
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
		Output() *Message
		// Rerror returns the call error.
		Rerror() *Rerror
		// Done returns the chan that indicates whether it has been completed.
		Done() <-chan struct{}
		// Reply returns the call reply.
		// Notes:
		//  Inside, <-Done() is automatically called and blocked,
		//  until the call is completed!
		Reply() (interface{}, *Rerror)
		// InputBodyCodec gets the body codec type of the input message.
		// Notes:
		//  Inside, <-Done() is automatically called and blocked,
		//  until the call is completed!
		InputBodyCodec() byte
		// InputMeta returns the header metadata of input message.
		// Notes:
		//  Inside, <-Done() is automatically called and blocked,
		//  until the call is completed!
		InputMeta() *utils.Args
		// CostTime returns the called cost time.
		// If PeerConfig.CountTime=false, always returns 0.
		// Notes:
		//  Inside, <-Done() is automatically called and blocked,
		//  until the call is completed!
		CostTime() time.Duration
	}
	callCmd struct {
		sess           *session
		output         *Message
		result         interface{}
		rerr           *Rerror
		inputBodyCodec byte
		inputMeta      *utils.Args
		start          time.Time
		cost           time.Duration
		swap           goutil.Map
		mu             sync.Mutex

		// Send itself to the public channel when call is complete.
		callCmdChan chan<- CallCmd
		// Strobes when call is complete.
		doneChan chan struct{}
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
	return c.Session(), true
}

// Session returns the session.
func (c *callCmd) Session() Session {
	return c.sess
}

// Ip returns the remote addr.
func (c *callCmd) Ip() string {
	return c.sess.RemoteAddr().String()
}

// RealIp returns the the current real remote addr.
func (c *callCmd) RealIp() string {
	realIp := c.inputMeta.Peek(MetaRealIp)
	if len(realIp) > 0 {
		return string(realIp)
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
func (c *callCmd) Output() *Message {
	return c.output
}

// Context carries a deadline, a cancelation signal, and other values across
// API boundaries.
func (c *callCmd) Context() context.Context {
	return c.output.Context()
}

// Rerror returns the call error.
func (c *callCmd) Rerror() *Rerror {
	return c.rerr
}

// Done returns the chan that indicates whether it has been completed.
func (c *callCmd) Done() <-chan struct{} {
	return c.doneChan
}

// Reply returns the call reply.
// Notes:
//  Inside, <-Done() is automatically called and blocked,
//  until the call is completed!
func (c *callCmd) Reply() (interface{}, *Rerror) {
	<-c.Done()
	return c.result, c.rerr
}

// InputBodyCodec gets the body codec type of the input message.
// Notes:
//  Inside, <-Done() is automatically called and blocked,
//  until the call is completed!
func (c *callCmd) InputBodyCodec() byte {
	<-c.Done()
	return c.inputBodyCodec
}

// InputMeta returns the header metadata of input message.
// Notes:
//  Inside, <-Done() is automatically called and blocked,
//  until the call is completed!
func (c *callCmd) InputMeta() *utils.Args {
	<-c.Done()
	return c.inputMeta
}

// CostTime returns the called cost time.
// If PeerConfig.CountTime=false, always returns 0.
// Notes:
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

func (c *callCmd) cancel() {
	c.sess.callCmdMap.Delete(c.output.Seq())
	c.rerr = rerrConnClosed
	c.callCmdChan <- c
	close(c.doneChan)
	// free count call-launch
	c.sess.graceCallCmdWaitGroup.Done()
}

// if callCmd.inputMeta!=nil, means the callCmd is replyed.
func (c *callCmd) hasReply() bool {
	return c.inputMeta != nil
}
