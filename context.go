// Copyright 2016 HenryLee. All Rights Reserved.
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
	"github.com/henrylee2cn/goutil"
)

// ConnCtx a context of Conn for one request/response.
type ConnCtx struct {
	Conn
	ctxPublic goutil.Map
}

var _ Conn = new(ConnCtx)

// NewConnCtx creates a context of Conn for one request/response.
func NewConnCtx(conn Conn) *ConnCtx {
	ctx := &ConnCtx{
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
func (ctx *ConnCtx) Public() goutil.Map {
	return ctx.ctxPublic
}

// PublicLen returns the length of public data of Conn Context.
func (ctx *ConnCtx) PublicLen() int {
	return ctx.ctxPublic.Len()
}

// SvrContext context of server
type SvrContext struct {
	*ConnCtx
}

// NewSvrContext creates a context of server.
func NewSvrContext(conn Conn) *SvrContext {
	svrCtx := &SvrContext{
		ConnCtx: NewConnCtx(conn),
	}
	return svrCtx
}

// CliContext context of client
type CliContext struct {
	*ConnCtx
}

// NewCliContext creates a context of client.
func NewCliContext(conn Conn) *CliContext {
	cliCtx := &CliContext{
		ConnCtx: NewConnCtx(conn),
	}
	return cliCtx
}
