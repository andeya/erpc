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

	"github.com/henrylee2cn/goutil"

	"github.com/henrylee2cn/teleport/socket"
)

// ApiType api type info
type ApiType struct {
	name            string
	originStruct    reflect.Type
	method          reflect.Method
	arg             reflect.Type
	reply           reflect.Type // only for api doc
	pluginContainer PluginContainer
}

// Context server controller ApiContext.
// For example:
//  type Home struct{ Context }
type (
	Context interface {
		Uri() string
		Path() string
		Query() url.Values
		Public() goutil.Map
		PublicLen() int
		SetBodyCodec(string)
		Ip() string
		// RealIp() string
	}
	ApiContext struct {
		session      *Session
		input        *socket.Packet
		output       *socket.Packet
		apiType      *ApiType
		originStruct reflect.Value
		method       reflect.Method
		arg          reflect.Value
		uri          *url.URL
		query        url.Values
		public       goutil.Map
		next         *ApiContext
	}
)

var _ Context = new(ApiContext)

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
	c.public = nil
	c.uri = nil
	c.query = nil
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

func (c *ApiContext) Uri() string {
	return c.input.Header.Uri
}

func (c *ApiContext) Path() string {
	return c.uri.Path
}

func (c *ApiContext) Query() url.Values {
	if c.query == nil {
		c.query = c.uri.Query()
	}
	return c.query
}

func (c *ApiContext) SetBodyCodec(codecName string) {
	c.output.BodyCodec = codecName
}

func (c *ApiContext) Ip() string {
	return c.session.socket.RemoteAddr().String()
}

func (c *ApiContext) binding(header *socket.Header) interface{} {
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

// TODO
func (c *ApiContext) bindResponse(header *socket.Header) interface{} {
	c.session.pullCmdMap.Load(header.Seq)
	return nil
}

func (c *ApiContext) bindPush(header *socket.Header) interface{} {
	var err error
	c.uri, err = url.Parse(header.Uri)
	if err != nil {
		return nil
	}
	var ok bool
	c.apiType, ok = c.session.apiMap.get(c.Path())
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
	c.apiType, ok = c.session.apiMap.get(c.Path())
	if !ok {
		c.output.Header.StatusCode = StatusNotFound
		c.output.Header.Status = StatusText(StatusNotFound)
		return nil
	}
	c.arg = reflect.New(c.apiType.arg)
	return c.arg.Interface()
}

// autoHandle handles request and push packet.
func (c *ApiContext) autoHandle() {
	rets := c.apiType.method.Func.Call([]reflect.Value{c.arg})
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
}
