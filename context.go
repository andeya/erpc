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
	// PreCtx context method set used before reading packet header.
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
	}
	// WriteCtx context method set for writing packet.
	WriteCtx interface {
		PreCtx
		// Output returns writed packet.
		Output() *socket.Packet
		// Rerror returns the handle error.
		Rerror() *Rerror
	}
	// inputCtx common context method set.
	inputCtx interface {
		PreCtx
		// Seq returns the input packet sequence.
		Seq() string
		// PeekMeta peeks the header metadata for the input packet.
		PeekMeta(key string) []byte
		// VisitMeta calls f for each existing metadata.
		//
		// f must not retain references to key and value after returning.
		// Make key and/or value copies if you need storing them after returning.
		VisitMeta(f func(key, value []byte))
		// CopyMeta returns the input packet metadata copy.
		CopyMeta() *utils.Args
		// Uri returns the input packet uri.
		Uri() string
		// UriObject returns the input packet uri object.
		UriObject() *url.URL
		// ResetUri resets the input packet uri.
		ResetUri(string)
		// Path returns the input packet uri path.
		Path() string
		// Query returns the input packet uri query object.
		Query() url.Values
	}
	// ReadCtx context method set for reading packet.
	ReadCtx interface {
		inputCtx
		// Input returns readed packet.
		Input() *socket.Packet
		// Rerror returns the handle error.
		Rerror() *Rerror
	}
	// PushCtx context method set for handling the pushed packet.
	// For example:
	//  type HomePush struct{ PushCtx }
	PushCtx interface {
		inputCtx
		// GetBodyCodec gets the body codec type of the input packet.
		GetBodyCodec() byte
	}
	// PullCtx context method set for handling the pulled packet.
	// For example:
	//  type HomePull struct{ PullCtx }
	PullCtx interface {
		inputCtx
		// Input returns readed packet.
		Input() *socket.Packet
		// GetBodyCodec gets the body codec type of the input packet.
		GetBodyCodec() byte
		// Output returns writed packet.
		Output() *socket.Packet
		// SetBodyCodec sets the body codec for reply packet.
		SetBodyCodec(byte)
		// AddMeta adds the header metadata 'key=value' for reply packet.
		// Multiple values for the same key may be added.
		AddMeta(key, value string)
		// SetMeta sets the header metadata 'key=value' for reply packet.
		SetMeta(key, value string)
		// AddXferPipe appends transfer filter pipe of reply packet.
		AddXferPipe(filterId ...byte)
	}
	// UnknownPushCtx context method set for handling the unknown pushed packet.
	UnknownPushCtx interface {
		inputCtx
		// GetBodyCodec gets the body codec type of the input packet.
		GetBodyCodec() byte
		// InputBodyBytes if the input body binder is []byte type, returns it, else returns nil.
		InputBodyBytes() []byte
		// Bind when the raw body binder is []byte type, now binds the input body to v.
		Bind(v interface{}) (bodyCodec byte, err error)
	}
	// UnknownPullCtx context method set for handling the unknown pulled packet.
	UnknownPullCtx interface {
		inputCtx
		// GetBodyCodec gets the body codec type of the input packet.
		GetBodyCodec() byte
		// InputBodyBytes if the input body binder is []byte type, returns it, else returns nil.
		InputBodyBytes() []byte
		// Bind when the raw body binder is []byte type, now binds the input body to v.
		Bind(v interface{}) (bodyCodec byte, err error)
		// SetBodyCodec sets the body codec for reply packet.
		SetBodyCodec(byte)
		// AddMeta adds the header metadata 'key=value' for reply packet.
		// Multiple values for the same key may be added.
		AddMeta(key, value string)
		// SetMeta sets the header metadata 'key=value' for reply packet.
		SetMeta(key, value string)
		// AddXferPipe appends transfer filter pipe of reply packet.
		AddXferPipe(filterId ...byte)
	}
)

var (
	_ PreCtx         = new(handlerCtx)
	_ inputCtx       = new(handlerCtx)
	_ WriteCtx       = new(handlerCtx)
	_ ReadCtx        = new(handlerCtx)
	_ PushCtx        = new(handlerCtx)
	_ PullCtx        = new(handlerCtx)
	_ UnknownPushCtx = new(handlerCtx)
	_ UnknownPullCtx = new(handlerCtx)
)

// handlerCtx the underlying common instance of PullCtx and PushCtx.
type handlerCtx struct {
	sess            *session
	input           *socket.Packet
	output          *socket.Packet
	handler         *Handler
	arg             reflect.Value
	pullCmd         *pullCmd
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
	c.input = socket.NewPacket()
	c.input.SetNewBody(c.binding)
	c.output = socket.NewPacket()
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
	c.pullCmd = nil
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

// Input returns readed packet.
func (c *handlerCtx) Input() *socket.Packet {
	return c.input
}

// Output returns writed packet.
func (c *handlerCtx) Output() *socket.Packet {
	return c.output
}

// Swap returns custom data swap of context.
func (c *handlerCtx) Swap() goutil.Map {
	return c.swap
}

// Seq returns the input packet sequence.
func (c *handlerCtx) Seq() string {
	return c.input.Seq()
}

// Uri returns the input packet uri string.
func (c *handlerCtx) Uri() string {
	return c.input.Uri()
}

// UriObject returns the input packet uri object.
func (c *handlerCtx) UriObject() *url.URL {
	return c.input.UriObject()
}

// ResetUri resets the input packet uri.
func (c *handlerCtx) ResetUri(uri string) {
	c.input.SetUri(uri)
}

// Path returns the input packet uri path.
func (c *handlerCtx) Path() string {
	return c.input.UriObject().Path
}

// Query returns the input packet uri query object.
func (c *handlerCtx) Query() url.Values {
	return c.input.UriObject().Query()
}

// PeekMeta peeks the header metadata for the input packet.
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

// CopyMeta returns the input packet metadata copy.
func (c *handlerCtx) CopyMeta() *utils.Args {
	dst := utils.AcquireArgs()
	c.input.Meta().CopyTo(dst)
	return dst
}

// AddMeta adds the header metadata 'key=value' for reply packet.
// Multiple values for the same key may be added.
func (c *handlerCtx) AddMeta(key, value string) {
	c.output.Meta().Add(key, value)
}

// SetMeta sets the header metadata 'key=value' for reply packet.
func (c *handlerCtx) SetMeta(key, value string) {
	c.output.Meta().Set(key, value)
}

// GetBodyCodec gets the body codec type of the input packet.
func (c *handlerCtx) GetBodyCodec() byte {
	return c.input.BodyCodec()
}

// SetBodyCodec sets the body codec for reply packet.
func (c *handlerCtx) SetBodyCodec(bodyCodec byte) {
	c.output.SetBodyCodec(bodyCodec)
}

// AddXferPipe appends transfer filter pipe of reply packet.
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

// Be executed synchronously when reading packet
func (c *handlerCtx) binding(header socket.Header) (body interface{}) {
	c.start = c.sess.timeNow()
	c.pluginContainer = c.sess.peer.pluginContainer
	switch header.Ptype() {
	case TypeReply:
		return c.bindReply(header)
	case TypePush:
		return c.bindPush(header)
	case TypePull:
		return c.bindPull(header)
	default:
		c.handleErr = rerrCodePtypeNotAllowed
		return nil
	}
}

const logFormatDisconnected = "disconnected due to unsupported packet type: %d\n%s %s %q\nRECV(%s)"

// Be executed asynchronously after readed packet
func (c *handlerCtx) handle() {
	if c.handleErr != nil && c.handleErr.Code == CodePtypeNotAllowed {
		goto E
	}
	switch c.input.Ptype() {
	case TypeReply:
		// handles pull reply
		c.handleReply()
		return

	case TypePush:
		//  handles push
		c.handlePush()
		return

	case TypePull:
		// handles and replies pull
		c.handlePull()
		return

	default:
	}
E:
	// if unsupported, disconnected.
	rerrCodePtypeNotAllowed.SetToMeta(c.output.Meta())
	Errorf(logFormatDisconnected, c.input.Ptype(), c.Ip(), c.input.Uri(), c.input.Seq(), packetLogBytes(c.input, c.sess.peer.printDetail))
	go c.sess.Close()
}

func (c *handlerCtx) bindPush(header socket.Header) interface{} {
	c.handleErr = c.pluginContainer.postReadPushHeader(c)
	if c.handleErr != nil {
		return nil
	}

	u := header.UriObject()
	if len(u.Path) == 0 {
		c.handleErr = rerrBadPacket.Copy().SetDetail("invalid URI for packet")
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
		c.sess.runlog(c.RealIp(), c.cost, c.input, nil, typePushHandle)
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

func (c *handlerCtx) bindPull(header socket.Header) interface{} {
	c.handleErr = c.pluginContainer.postReadPullHeader(c)
	if c.handleErr != nil {
		return nil
	}

	u := header.UriObject()
	if len(u.Path) == 0 {
		c.handleErr = rerrBadPacket.Copy().SetDetail("invalid URI for packet")
		return nil
	}

	var ok bool
	c.handler, ok = c.sess.getPullHandler(u.Path)
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

	c.handleErr = c.pluginContainer.preReadPullBody(c)
	if c.handleErr != nil {
		return nil
	}

	return c.input.Body()
}

// handlePull handles and replies pull.
func (c *handlerCtx) handlePull() {
	var writed bool
	defer func() {
		if p := recover(); p != nil {
			Debugf("panic:%v\n%s", p, goutil.PanicTrace(2))
			if !writed {
				if c.handleErr == nil {
					c.handleErr = rerrInternalServerError.Copy().SetDetail(fmt.Sprint(p))
				}
				c.writeReply(c.handleErr)
			}
		}
		c.cost = c.sess.timeSince(c.start)
		c.sess.runlog(c.RealIp(), c.cost, c.input, c.output, typePullHandle)
	}()

	c.output.SetPtype(TypeReply)
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

	// handle pull
	if c.handleErr == nil {
		c.handleErr = c.pluginContainer.postReadPullBody(c)
		if c.handleErr == nil {
			if c.handler.isUnknown {
				c.handler.unknownHandleFunc(c)
			} else {
				c.handler.handleFunc(c, c.arg)
			}
		}
	}

	// reply pull
	c.setReplyBodyCodec(c.handleErr != nil)
	c.pluginContainer.preWriteReply(c)
	rerr := c.writeReply(c.handleErr)
	if rerr != nil {
		if c.handleErr == nil {
			c.handleErr = rerr
		}
		if rerr != rerrConnClosed {
			c.writeReply(rerrInternalServerError.Copy().SetDetail(rerr.Detail))
		}
		return
	}
	writed = true
	c.pluginContainer.postWriteReply(c)
}

func (c *handlerCtx) setReplyBodyCodec(hasError bool) {
	if hasError {
		return
	}
	if c.output.BodyCodec() != codec.NilCodecId {
		return
	}
	acceptBodyCodec, ok := GetAcceptBodyCodec(c.input.Meta())
	if ok {
		if _, err := codec.Get(acceptBodyCodec); err == nil {
			c.output.SetBodyCodec(acceptBodyCodec)
			return
		}
	}
	c.output.SetBodyCodec(c.input.BodyCodec())
}

func (c *handlerCtx) writeReply(rerr *Rerror) *Rerror {
	if rerr != nil {
		rerr.SetToMeta(c.output.Meta())
		c.output.SetBody(nil)
		c.output.SetBodyCodec(codec.NilCodecId)
		_, rerr = c.sess.write(c.output)
		return rerr
	}
	_, rerr = c.sess.write(c.output)
	return rerr
}

func (c *handlerCtx) bindReply(header socket.Header) interface{} {
	_pullCmd, ok := c.sess.pullCmdMap.Load(header.Seq())
	if !ok {
		Warnf("not found pull cmd: %v", c.input)
		return nil
	}
	c.pullCmd = _pullCmd.(*pullCmd)

	// unlock: handleReply
	c.pullCmd.mu.Lock()

	c.swap = c.pullCmd.swap
	c.pullCmd.inputBodyCodec = c.GetBodyCodec()
	// if c.pullCmd.inputMeta!=nil, means the pullCmd is replyed.
	c.input.Meta().CopyTo(c.pullCmd.inputMeta)
	c.setContext(c.pullCmd.output.Context())
	c.input.SetBody(c.pullCmd.result)

	rerr := c.pluginContainer.postReadReplyHeader(c)
	if rerr != nil {
		c.pullCmd.rerr = rerr
		return nil
	}
	rerr = c.pluginContainer.preReadReplyBody(c)
	if rerr != nil {
		c.pullCmd.rerr = rerr
		return nil
	}
	return c.input.Body()
}

// handleReply handles pull reply.
func (c *handlerCtx) handleReply() {
	if c.pullCmd == nil {
		return
	}

	// lock: bindReply
	defer c.pullCmd.mu.Unlock()

	defer func() {
		if p := recover(); p != nil {
			Debugf("panic:%v\n%s", p, goutil.PanicTrace(2))
		}
		c.pullCmd.result = c.input.Body()
		c.handleErr = c.pullCmd.rerr
		c.pullCmd.done()
		c.pullCmd.cost = c.sess.timeSince(c.pullCmd.start)
		c.sess.runlog(c.RealIp(), c.pullCmd.cost, c.input, c.pullCmd.output, typePullLaunch)
	}()
	if c.pullCmd.rerr != nil {
		return
	}
	rerr := NewRerrorFromMeta(c.input.Meta())
	if rerr == nil {
		rerr = c.pluginContainer.postReadReplyBody(c)
	}
	c.pullCmd.rerr = rerr
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
	// PullCmd the command of the pulling operation's response.
	PullCmd interface {
		// Context carries a deadline, a cancelation signal, and other values across
		// API boundaries.
		Context() context.Context
		// Output returns writed packet.
		Output() *socket.Packet
		// Rerror returns the pull error.
		Rerror() *Rerror
		// Done returns the chan that indicates whether it has been completed.
		Done() <-chan struct{}
		// Reply returns the pull reply.
		// Notes:
		//  Inside, <-Done() is automatically called and blocked,
		//  until the pull is completed!
		Reply() (interface{}, *Rerror)
		// InputBodyCodec gets the body codec type of the input packet.
		// Notes:
		//  Inside, <-Done() is automatically called and blocked,
		//  until the pull is completed!
		InputBodyCodec() byte
		// InputMeta returns the header metadata of input packet.
		// Notes:
		//  Inside, <-Done() is automatically called and blocked,
		//  until the pull is completed!
		InputMeta() *utils.Args
		// CostTime returns the pulled cost time.
		// If PeerConfig.CountTime=false, always returns 0.
		// Notes:
		//  Inside, <-Done() is automatically called and blocked,
		//  until the pull is completed!
		CostTime() time.Duration
	}
	pullCmd struct {
		sess           *session
		output         *socket.Packet
		result         interface{}
		rerr           *Rerror
		inputBodyCodec byte
		inputMeta      *utils.Args
		start          time.Time
		cost           time.Duration
		swap           goutil.Map
		mu             sync.Mutex

		// Send itself to the public channel when pull is complete.
		pullCmdChan chan<- PullCmd
		// Strobes when pull is complete.
		doneChan chan struct{}
	}
)

var _ WriteCtx = new(pullCmd)

// Peer returns the peer.
func (p *pullCmd) Peer() Peer {
	return p.sess.peer
}

// Session returns the session.
func (p *pullCmd) Session() Session {
	return p.sess
}

// Ip returns the remote addr.
func (p *pullCmd) Ip() string {
	return p.sess.RemoteAddr().String()
}

// RealIp returns the the current real remote addr.
func (p *pullCmd) RealIp() string {
	realIp := p.inputMeta.Peek(MetaRealIp)
	if len(realIp) > 0 {
		return string(realIp)
	}
	return p.sess.RemoteAddr().String()
}

// Swap returns custom data swap of context.
func (p *pullCmd) Swap() goutil.Map {
	return p.swap
}

// SwapLen returns the amount of recorded custom data of context.
func (p *pullCmd) SwapLen() int {
	return p.swap.Len()
}

// Output returns writed packet.
func (p *pullCmd) Output() *socket.Packet {
	return p.output
}

// Context carries a deadline, a cancelation signal, and other values across
// API boundaries.
func (p *pullCmd) Context() context.Context {
	return p.output.Context()
}

// Rerror returns the pull error.
func (p *pullCmd) Rerror() *Rerror {
	return p.rerr
}

// Done returns the chan that indicates whether it has been completed.
func (p *pullCmd) Done() <-chan struct{} {
	return p.doneChan
}

// Reply returns the pull reply.
// Notes:
//  Inside, <-Done() is automatically called and blocked,
//  until the pull is completed!
func (p *pullCmd) Reply() (interface{}, *Rerror) {
	<-p.Done()
	return p.result, p.rerr
}

// InputBodyCodec gets the body codec type of the input packet.
// Notes:
//  Inside, <-Done() is automatically called and blocked,
//  until the pull is completed!
func (p *pullCmd) InputBodyCodec() byte {
	<-p.Done()
	return p.inputBodyCodec
}

// InputMeta returns the header metadata of input packet.
// Notes:
//  Inside, <-Done() is automatically called and blocked,
//  until the pull is completed!
func (p *pullCmd) InputMeta() *utils.Args {
	<-p.Done()
	return p.inputMeta
}

// CostTime returns the pulled cost time.
// If PeerConfig.CountTime=false, always returns 0.
// Notes:
//  Inside, <-Done() is automatically called and blocked,
//  until the pull is completed!
func (p *pullCmd) CostTime() time.Duration {
	<-p.Done()
	return p.cost
}

func (p *pullCmd) done() {
	p.sess.pullCmdMap.Delete(p.output.Seq())
	p.pullCmdChan <- p
	close(p.doneChan)
	// free count pull-launch
	p.sess.gracePullCmdWaitGroup.Done()
}

func (p *pullCmd) cancel() {
	p.sess.pullCmdMap.Delete(p.output.Seq())
	p.rerr = rerrConnClosed
	p.pullCmdChan <- p
	close(p.doneChan)
	// free count pull-launch
	p.sess.gracePullCmdWaitGroup.Done()
}

// if pullCmd.inputMeta!=nil, means the pullCmd is replyed.
func (p *pullCmd) hasReply() bool {
	return p.inputMeta != nil
}
