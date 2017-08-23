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
	"compress/gzip"
	"net/url"
	"reflect"
	"time"

	"github.com/henrylee2cn/goutil"

	"github.com/henrylee2cn/teleport/socket"
)

type (
	// PullCtx request handler context.
	// For example:
	//  type HomePull struct{ PullCtx }
	PullCtx interface {
		PushCtx
		SetBodyCodec(string)
	}
	// PushCtx push handler context.
	// For example:
	//  type HomePush struct{ PushCtx }
	PushCtx interface {
		Uri() string
		Path() string
		Query() url.Values
		Public() goutil.Map
		PublicLen() int
		Ip() string
		Peer() *Peer
		Session() *Session
	}
	UnknowPullCtx interface {
		PullCtx
		Unmarshal(b []byte, v interface{}, assignToInput ...bool) (codecName string, err error)
	}
	UnknowPushCtx interface {
		PushCtx
		Unmarshal(b []byte, v interface{}, assignToInput ...bool) (codecName string, err error)
	}
	// WriteCtx for writing packet.
	WriteCtx interface {
		Public() goutil.Map
		PublicLen() int
		Ip() string
		Peer() *Peer
		Session() *Session
		Output() *socket.Packet
	}
	// ReadCtx for reading packet.
	ReadCtx interface {
		Public() goutil.Map
		PublicLen() int
		Ip() string
		Peer() *Peer
		Session() *Session
		Input() *socket.Packet
	}
)

// readHandleCtx the underlying common instance of PullCtx and PushCtx.
type readHandleCtx struct {
	session           *Session
	input             *socket.Packet
	output            *socket.Packet
	apiType           *Handler
	originStructMaker func(*readHandleCtx) reflect.Value
	method            reflect.Method
	arg               reflect.Value
	pullCmd           *PullCmd
	uri               *url.URL
	query             url.Values
	public            goutil.Map
	start             time.Time
	cost              time.Duration
	pluginContainer   PluginContainer
	next              *readHandleCtx
}

var (
	_ PullCtx       = new(readHandleCtx)
	_ PushCtx       = new(readHandleCtx)
	_ WriteCtx      = new(readHandleCtx)
	_ ReadCtx       = new(readHandleCtx)
	_ UnknowPullCtx = new(readHandleCtx)
	_ UnknowPushCtx = new(readHandleCtx)
)

// newReadHandleCtx creates a readHandleCtx for one request/response or push.
func newReadHandleCtx() *readHandleCtx {
	c := new(readHandleCtx)
	c.input = socket.NewPacket(c.binding)
	c.output = socket.NewPacket(nil)
	return c
}

func (c *readHandleCtx) reInit(s *Session) {
	c.session = s
	c.public = goutil.RwMap()
	if s.socket.PublicLen() > 0 {
		s.socket.Public().Range(func(key, value interface{}) bool {
			c.public.Store(key, value)
			return true
		})
	}
}

var (
	emptyValue  = reflect.Value{}
	emptyMethod = reflect.Method{}
)

func (c *readHandleCtx) clean() {
	c.session = nil
	c.apiType = nil
	c.arg = emptyValue
	c.originStructMaker = nil
	c.method = emptyMethod
	c.pullCmd = nil
	c.public = nil
	c.uri = nil
	c.query = nil
	c.cost = 0
	c.pluginContainer = nil
	c.input.Reset(c.binding)
	c.output.Reset(nil)
}

// Peer returns the peer.
func (c *readHandleCtx) Peer() *Peer {
	return c.session.peer
}

// Session returns the session.
func (c *readHandleCtx) Session() *Session {
	return c.session
}

// Input returns readed packet.
func (c *readHandleCtx) Input() *socket.Packet {
	return c.input
}

// Output returns writed packet.
func (c *readHandleCtx) Output() *socket.Packet {
	return c.output
}

// Public returns temporary public data of context.
func (c *readHandleCtx) Public() goutil.Map {
	return c.public
}

// PublicLen returns the length of public data of context.
func (c *readHandleCtx) PublicLen() int {
	return c.public.Len()
}

// Uri returns the input packet uri.
func (c *readHandleCtx) Uri() string {
	return c.input.Header.Uri
}

// Path returns the input packet uri path.
func (c *readHandleCtx) Path() string {
	return c.uri.Path
}

// Query returns the input packet uri query.
func (c *readHandleCtx) Query() url.Values {
	if c.query == nil {
		c.query = c.uri.Query()
	}
	return c.query
}

// SetBodyCodec sets the body codec for response packet.
func (c *readHandleCtx) SetBodyCodec(codecName string) {
	c.output.BodyCodec = codecName
}

// Ip returns the remote addr.
func (c *readHandleCtx) Ip() string {
	return c.session.RemoteIp()
}

func (c *readHandleCtx) binding(header *socket.Header) (body interface{}) {
	defer func() {
		if p := recover(); p != nil {
			Errorf("panic:\n%v\n%s", p, goutil.PanicTrace(1))
			body = nil
		}
	}()
	c.start = time.Now()
	c.pluginContainer = c.session.peer.pluginContainer
	switch header.Type {
	case TypeReply:
		return c.bindReply(header)

	case TypePush:
		return c.bindPush(header)

	case TypePull:
		return c.bindPull(header)

	default:
		return nil
	}
}

func (c *readHandleCtx) bindPush(header *socket.Header) interface{} {
	err := c.pluginContainer.PostReadHeader(c)
	if err != nil {
		Errorf("%s", err.Error())
		return nil
	}

	c.uri, err = url.Parse(header.Uri)
	if err != nil {
		return nil
	}

	var ok bool
	c.apiType, ok = c.session.pushRouter.get(c.Path())
	if !ok {
		return nil
	}

	c.pluginContainer = c.apiType.pluginContainer
	c.originStructMaker = c.apiType.originStructMaker
	c.arg = reflect.New(c.apiType.argElem)
	c.input.Body = c.arg.Interface()

	err = c.pluginContainer.PreReadBody(c)
	if err != nil {
		Errorf("%s", err.Error())
		return nil
	}

	return c.input.Body
}

func (c *readHandleCtx) bindPull(header *socket.Header) interface{} {
	err := c.pluginContainer.PostReadHeader(c)
	if err != nil {
		errStr := err.Error()
		Errorf("%s", errStr)
		c.output.Header.StatusCode = StatusFailedPlugin
		c.output.Header.Status = errStr
		return nil
	}

	c.output.Header.Seq = c.input.Header.Seq
	c.output.Header.Type = TypeReply
	c.output.Header.Uri = c.input.Header.Uri
	c.output.HeaderCodec = c.input.HeaderCodec
	c.output.Header.Gzip = c.input.Header.Gzip

	c.uri, err = url.Parse(header.Uri)
	if err != nil {
		c.output.Header.StatusCode = StatusBadUri
		c.output.Header.Status = err.Error()
		return nil
	}

	var ok bool
	c.apiType, ok = c.session.pullRouter.get(c.Path())
	if !ok {
		c.output.Header.StatusCode = StatusNotFound
		c.output.Header.Status = StatusText(StatusNotFound)
		return nil
	}

	c.pluginContainer = c.apiType.pluginContainer
	c.originStructMaker = c.apiType.originStructMaker
	c.arg = reflect.New(c.apiType.argElem)
	c.input.Body = c.arg.Interface()

	err = c.pluginContainer.PreReadBody(c)
	if err != nil {
		errStr := err.Error()
		Errorf("%s", errStr)
		c.output.Header.StatusCode = StatusFailedPlugin
		c.output.Header.Status = errStr
		return nil
	}

	c.output.Header.StatusCode = StatusOK
	c.output.Header.Status = StatusText(StatusOK)
	return c.input.Body
}

// handle handles and replies pull, or handles push.
func (c *readHandleCtx) handle() {
	defer func() {
		if p := recover(); p != nil {
			Errorf("panic:\n%v\n%s", p, goutil.PanicTrace(1))
		}
		c.cost = time.Since(c.start)
		c.session.runlog(c.cost, c.input, c.output)
	}()
	err := c.pluginContainer.PostReadBody(c)
	if err != nil {
		errStr := err.Error()
		Errorf("%s", errStr)
		if isPullHandle(c.input, c.output) {
			c.output.Header.StatusCode = StatusFailedPlugin
			c.output.Header.Status = errStr
			c.output = nil
		} else {
			return
		}
	}

	var statusOK = c.output.Header.StatusCode == StatusOK
	if statusOK {
		rets := c.apiType.method.Call([]reflect.Value{c.originStructMaker(c), c.arg})

		// receive push
		if len(rets) == 0 {
			return
		}

		c.output.Body = rets[0].Interface()
		e, ok := rets[1].Interface().(Xerror)
		if ok && e != nil {
			c.output.Header.StatusCode = e.Code()
			c.output.Header.Status = e.Text()
		}
	}

	// reply pull
	if len(c.output.BodyCodec) == 0 {
		c.output.BodyCodec = c.input.BodyCodec
	}

	if err = c.pluginContainer.PreWriteReply(c); err != nil {
		errStr := err.Error()
		c.output.Body = nil
		if statusOK {
			c.output.Header.StatusCode = StatusFailedPlugin
			c.output.Header.Status = errStr
		}
		Errorf("%s", errStr)
	}

	if err = c.session.write(c.output); err != nil {
		Warnf("WritePacket: %s", err.Error())
	}

	if err = c.pluginContainer.PostWriteReply(c); err != nil {
		Errorf("%s", err.Error())
	}
}

func (c *readHandleCtx) bindReply(header *socket.Header) interface{} {
	pullCmd, ok := c.session.pullCmdMap.Load(header.Seq)
	if !ok {
		return nil
	}
	c.session.pullCmdMap.Delete(header.Seq)
	c.pullCmd = pullCmd.(*PullCmd)
	c.public = c.pullCmd.public

	err := c.pluginContainer.PostReadHeader(c)
	if err != nil {
		c.pullCmd.Xerror = NewXerror(StatusFailedPlugin, err.Error())
		return nil
	}
	err = c.pluginContainer.PreReadBody(c)
	if err != nil {
		c.pullCmd.Xerror = NewXerror(StatusFailedPlugin, err.Error())
		return nil
	}
	return c.pullCmd.reply
}

// handleReply handles pull reply.
func (c *readHandleCtx) handleReply() {
	err := c.pluginContainer.PostReadBody(c)
	if err != nil {
		c.pullCmd.Xerror = NewXerror(StatusFailedPlugin, err.Error())
	}

	c.pullCmd.done()
	c.pullCmd.cost = time.Since(c.pullCmd.start)
	c.session.runlog(c.pullCmd.cost, c.input, c.pullCmd.output)
}

// Unmarshal unmarshals bytes to header or body receiver.
func (c *readHandleCtx) Unmarshal(b []byte, v interface{}, assignToInput ...bool) (string, error) {
	codecName, err := socket.Unmarshal(b, v, c.input.Header.Gzip != gzip.NoCompression)
	if len(assignToInput) > 0 && assignToInput[0] {
		c.input.Body = v
		c.input.BodyCodec = codecName
	}
	return codecName, err
}

// PullCmd the command of the pulling operation's response.
type PullCmd struct {
	session  *Session
	output   *socket.Packet
	reply    interface{}
	doneChan chan *PullCmd // Strobes when pull is complete.
	start    time.Time
	cost     time.Duration
	public   goutil.Map
	Xerror   Xerror
}

var _ WriteCtx = new(PullCmd)

// Peer returns the peer.
func (c *PullCmd) Peer() *Peer {
	return c.session.peer
}

// Session returns the session.
func (c *PullCmd) Session() *Session {
	return c.session
}

// Ip returns the remote addr.
func (c *PullCmd) Ip() string {
	return c.session.RemoteIp()
}

// Public returns temporary public data of context.
func (c *PullCmd) Public() goutil.Map {
	return c.public
}

// PublicLen returns the length of public data of context.
func (c *PullCmd) PublicLen() int {
	return c.public.Len()
}

// Output returns writed packet.
func (c *PullCmd) Output() *socket.Packet {
	return c.output
}

func (p *PullCmd) done() {
	p.doneChan <- p
}
