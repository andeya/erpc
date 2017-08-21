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
	"net/url"
	"reflect"
	"time"

	"github.com/henrylee2cn/go-logging/color"
	"github.com/henrylee2cn/goutil"

	"github.com/henrylee2cn/teleport/socket"
)

type (
	// RequestCtx request handler context.
	// For example:
	//  type Home struct{ RequestCtx }
	RequestCtx interface {
		PushCtx
		SetBodyCodec(string)
	}
	// PushCtx push handler context.
	// For example:
	//  type Home struct{ PushCtx }
	PushCtx interface {
		Uri() string
		Path() string
		Query() url.Values
		Public() goutil.Map
		PublicLen() int
		Ip() string
		// RealIp() string
	}
	// ApiContext the underlying common instance of RequestCtx and PushCtx.
	ApiContext struct {
		session      *Session
		input        *socket.Packet
		output       *socket.Packet
		apiType      *Handler
		originStruct reflect.Value
		method       reflect.Method
		arg          reflect.Value
		pullCmd      *PullCmd
		uri          *url.URL
		query        url.Values
		public       goutil.Map
		start        time.Time
		cost         time.Duration
		next         *ApiContext
	}
)

var (
	_ RequestCtx = new(ApiContext)
	_ PushCtx    = new(ApiContext)
)

// newApiContext creates a ApiContext for one request/response or push.
func newApiContext() *ApiContext {
	c := new(ApiContext)
	c.input = socket.NewPacket(c.binding)
	c.output = socket.NewPacket(nil)
	return c
}

func (c *ApiContext) reInit(s *Session) {
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

func (c *ApiContext) clean() {
	c.session = nil
	c.apiType = nil
	c.arg = emptyValue
	c.originStruct = emptyValue
	c.method = emptyMethod
	c.pullCmd = nil
	c.public = nil
	c.uri = nil
	c.query = nil
	c.cost = 0
	c.input.Reset(c.binding)
	c.output.Reset(nil)
}

// Public returns temporary public data of Conn Context.
func (c *ApiContext) Public() goutil.Map {
	return c.public
}

// PublicLen returns the length of public data of Conn Context.
func (c *ApiContext) PublicLen() int {
	return c.public.Len()
}

// Uri returns the input packet uri.
func (c *ApiContext) Uri() string {
	return c.input.Header.Uri
}

// Path returns the input packet uri path.
func (c *ApiContext) Path() string {
	return c.uri.Path
}

// Query returns the input packet uri query.
func (c *ApiContext) Query() url.Values {
	if c.query == nil {
		c.query = c.uri.Query()
	}
	return c.query
}

// SetBodyCodec sets the body codec for response packet.
func (c *ApiContext) SetBodyCodec(codecName string) {
	c.output.BodyCodec = codecName
}

// Ip returns the remote addr.
func (c *ApiContext) Ip() string {
	return c.session.socket.RemoteAddr().String()
}

func (c *ApiContext) binding(header *socket.Header) interface{} {
	c.start = time.Now()
	switch header.Type {
	case TypeResponse:
		return c.bindResponse(header)

	case TypePush:
		return c.bindPush(header)

	case TypeRequest:
		return c.bindRequest(header)

	default:
		return nil
	}
}

func (c *ApiContext) bindResponse(header *socket.Header) interface{} {
	pullCmd, ok := c.session.pullCmdMap.Load(header.Seq)
	if !ok {
		return nil
	}
	c.session.pullCmdMap.Delete(header.Seq)
	c.pullCmd = pullCmd.(*PullCmd)
	return c.pullCmd.reply
}

func (c *ApiContext) bindPush(header *socket.Header) interface{} {
	var err error
	c.uri, err = url.Parse(header.Uri)
	if err != nil {
		return nil
	}
	var ok bool
	c.apiType, ok = c.session.pushRouter.get(c.Path())
	if !ok {
		return nil
	}
	c.arg = reflect.New(c.apiType.arg)
	return c.arg.Interface()
}

func (c *ApiContext) bindRequest(header *socket.Header) interface{} {
	c.output.Header.Seq = c.input.Header.Seq
	c.output.Header.Type = TypeResponse
	c.output.Header.Uri = c.input.Header.Uri
	c.output.HeaderCodec = c.input.HeaderCodec
	c.output.Header.Gzip = c.input.Header.Gzip

	var err error
	c.uri, err = url.Parse(header.Uri)
	if err != nil {
		c.output.Header.StatusCode = StatusBadRequest
		c.output.Header.Status = err.Error()
		return nil
	}
	var ok bool
	c.apiType, ok = c.session.requestRouter.get(c.Path())
	if !ok {
		c.output.Header.StatusCode = StatusNotFound
		c.output.Header.Status = StatusText(StatusNotFound)
		return nil
	}
	c.arg = reflect.New(c.apiType.arg)
	return c.arg.Interface()
}

// handle handles request and push packet.
func (c *ApiContext) handle() {
	rets := c.apiType.method.Func.Call([]reflect.Value{c.arg})
	if len(rets) == 0 {
		return
	}
	c.output.Body = rets[0].Interface()
	e := rets[0].Interface().(Xerror)
	if e == nil {
		c.output.Header.StatusCode = StatusOK
		c.output.Header.Status = StatusText(StatusOK)
	} else {
		c.output.Header.StatusCode = e.Code()
		c.output.Header.Status = e.Text()
	}
	if len(c.output.BodyCodec) == 0 {
		c.output.BodyCodec = c.input.BodyCodec
	}

	err := c.session.write(c.output)
	if err != nil {
		Debugf("teleport: WritePacket: %s", err.Error())
	}

	var n = c.output.Header.StatusCode
	var code string
	switch {
	case n >= 500:
		code = color.Red(n)
	case n >= 400:
		code = color.Magenta(n)
	case n >= 300:
		code = color.Grey(n)
	default:
		code = color.Green(n)
	}
	c.cost = time.Since(c.start)
	if c.cost < c.session.peer.slowCometDuration {
		Infof("%15s %3s %10d %12s %-30s | %v", c.Ip(), code, c.output.Length, c.cost, c.output.Header.Uri, c.output.Body)
	} else {
		Warnf(" %15s %3s %10d %12s(slow) %-30s | %v", c.Ip(), code, c.output.Length, c.cost, c.output.Header.Uri, c.output.Body)
	}
}

// respHandle handles request and push packet.
func (c *ApiContext) respHandle() {
	c.pullCmd.done()
}
