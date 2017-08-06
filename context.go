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
