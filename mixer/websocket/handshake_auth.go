package websocket

import (
	"github.com/henrylee2cn/erpc/v6"
	ws "github.com/henrylee2cn/erpc/v6/mixer/websocket/websocket"
	"net/http"
)

func NewHandshakeAuthPlugin(ckFn Checker, apFn Acceptor) *HandshakeAuthPlugin {
	return &HandshakeAuthPlugin{
		CheckFunc:  ckFn,
		AcceptFunc: apFn,
	}
}

type Acceptor func(sess erpc.Session) *erpc.Status
type Checker func(r *http.Request) (sessionId string, status *erpc.Status)

type HandshakeAuthPlugin struct {
	CheckFunc  Checker
	AcceptFunc Acceptor
}

var (
	_ PostWebsocketAcceptPlugin   = new(HandshakeAuthPlugin)
	_ PreWebsocketHandshakePlugin = new(HandshakeAuthPlugin)
)

func (t *HandshakeAuthPlugin) Name() string {
	return "handshake-auth-plugin"
}

const sessionHeader = "Session-Id"

func (p *HandshakeAuthPlugin) PreHandshake(r *http.Request) *erpc.Status {
	if p.CheckFunc == nil {
		return nil
	}
	id, stat := p.CheckFunc(r)
	r.Header.Set(sessionHeader, id)
	return stat
}

func (p *HandshakeAuthPlugin) PostAccept(sess erpc.Session, conn *ws.Conn) *erpc.Status {
	if p.AcceptFunc == nil {
		return nil
	}
	id := conn.Request().Header.Get(sessionHeader)
	sess.SetID(id)
	stat := p.AcceptFunc(sess)
	return stat
	//return erpc.NewStatus(erpc.CodeOK, erpc.CodeText(erpc.CodeOK))
}

func QueryToken(tokenKey string, r *http.Request) (token string) {
	queryParams := r.URL.Query()
	if values, ok := queryParams[tokenKey]; ok {
		token = values[0]
	}
	return token
}
