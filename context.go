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

	"github.com/henrylee2cn/goutil"
)

// SvrContext server controller context.
// For example:
//  type Home struct{ SvrContext }
type SvrContext interface {
	Uri() string
	Path() string
	Query() url.Values
	Public() goutil.Map
	PublicLen() int
	SetCodec(string)
	Ip() string
	// RealIp() string
}

// ConnCtx a context interface of Conn for one request/response.
type ConnCtx interface {
	Conn
}

type context struct {
	Conn
	ctxPublic  goutil.Map
	reqHeader  *Header
	reqUri     *url.URL
	reqQuery   url.Values
	respHeader *Header
}

var (
	_ SvrContext = new(context)
	_ ConnCtx    = new(context)
)

// newCtx creates a context of Conn for one request/response.
func newCtx(conn Conn) *context {
	ctx := &context{
		Conn:      conn,
		ctxPublic: goutil.NormalMap(),
	}
	if conn.PublicLen() > 0 {
		conn.Public().Range(func(key, value interface{}) bool {
			ctx.ctxPublic.Store(key, value)
			return true
		})
	}
	return ctx
}

// Public returns temporary public data of Conn Context.
func (c *context) Public() goutil.Map {
	return c.ctxPublic
}

// PublicLen returns the length of public data of Conn Context.
func (c *context) PublicLen() int {
	return c.ctxPublic.InexactLen()
}

func (c *context) Uri() string {
	return c.reqHeader.GetUri()
}

func (c *context) getUri() *url.URL {
	if c.reqUri == nil {
		c.reqUri, _ = url.Parse(c.Uri())
	}
	return c.reqUri
}

func (c *context) Path() string {
	return c.getUri().Path
}

func (c *context) Query() url.Values {
	if c.reqQuery == nil {
		c.reqQuery = c.getUri().Query()
	}
	return c.reqQuery
}

func (c *context) SetCodec(codec string) {
	c.respHeader.Codec = codec
}

func (c *context) Ip() string {
	return c.Conn.RemoteAddr().String()
}
