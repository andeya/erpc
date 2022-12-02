package websocket_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/andeya/erpc/v7"
	ws "github.com/andeya/erpc/v7/mixer/websocket"
	"github.com/andeya/erpc/v7/mixer/websocket/jsonSubProto"
	"github.com/andeya/erpc/v7/mixer/websocket/pbSubProto"
	"github.com/andeya/erpc/v7/plugin/auth"
)

type Arg struct {
	A int
	B int `param:"<range:1:>"`
}

type P struct{ erpc.CallCtx }

func (p *P) Divide(arg *Arg) (int, *erpc.Status) {
	return arg.A / arg.B, nil
}

func TestJSONWebsocket(t *testing.T) {
	srv := ws.NewServer("/", erpc.PeerConfig{ListenPort: 9090})
	srv.RouteCall(new(P))
	go srv.ListenAndServe()

	time.Sleep(time.Second * 1)

	cli := ws.NewClient("/", erpc.PeerConfig{})
	sess, stat := cli.Dial(":9090")
	if !stat.OK() {
		t.Fatal(stat)
	}
	var result int
	stat = sess.Call("/p/divide", &Arg{
		A: 10,
		B: 2,
	}, &result,
	).Status()
	if !stat.OK() {
		t.Fatal(stat)
	}
	t.Logf("10/2=%d", result)
	time.Sleep(time.Second)
}

func TestPbWebsocketTLS(t *testing.T) {
	srv := ws.NewServer("/abc", erpc.PeerConfig{ListenPort: 9091})
	srv.RouteCall(new(P))
	srv.SetTLSConfig(erpc.GenerateTLSConfigForServer())
	go srv.ListenAndServeProtobuf()

	time.Sleep(time.Second * 1)

	cli := ws.NewClient("/abc", erpc.PeerConfig{})
	cli.SetTLSConfig(erpc.GenerateTLSConfigForClient())
	sess, err := cli.DialProtobuf(":9091")
	if err != nil {
		t.Fatal(err)
	}
	var result int
	stat := sess.Call("/p/divide", &Arg{
		A: 10,
		B: 2,
	}, &result,
	).Status()
	if !stat.OK() {
		t.Fatal(stat)
	}
	t.Logf("10/2=%d", result)
	time.Sleep(time.Second)
}

func TestCustomizedWebsocket(t *testing.T) {
	srv := erpc.NewPeer(erpc.PeerConfig{})
	http.Handle("/ws", ws.NewPbServeHandler(srv, nil))
	go http.ListenAndServe(":9092", nil)
	srv.RouteCall(new(P))
	time.Sleep(time.Second * 1)

	cli := erpc.NewPeer(erpc.PeerConfig{}, ws.NewDialPlugin("/ws"))
	sess, stat := cli.Dial(":9092", pbSubProto.NewPbSubProtoFunc())
	if !stat.OK() {
		t.Fatal(stat)
	}
	var result int
	stat = sess.Call("/p/divide", &Arg{
		A: 10,
		B: 2,
	}, &result,
	).Status()
	if !stat.OK() {
		t.Fatal(stat)
	}
	t.Logf("10/2=%d", result)
	time.Sleep(time.Second)
}

func TestJSONWebsocketAuth(t *testing.T) {
	srv := ws.NewServer(
		"/auth",
		erpc.PeerConfig{ListenPort: 9093},
		authChecker,
	)
	srv.RouteCall(new(P))
	go srv.ListenAndServe()

	time.Sleep(time.Second * 1)

	cli := ws.NewClient(
		"/auth",
		erpc.PeerConfig{},
		authBearer,
	)
	sess, stat := cli.Dial(":9093")
	if !stat.OK() {
		t.Fatal(stat)
	}
	var result int
	stat = sess.Call("/p/divide", &Arg{
		A: 10,
		B: 2,
	}, &result,
	).Status()
	if !stat.OK() {
		t.Fatal(stat)
	}
	t.Logf("10/2=%d", result)
	time.Sleep(time.Second)
}

const clientAuthInfo = "client-auth-info-12345"

var authBearer = auth.NewBearerPlugin(
	func(sess auth.Session, fn auth.SendOnce) (stat *erpc.Status) {
		var ret string
		stat = fn(clientAuthInfo, &ret)
		if !stat.OK() {
			return
		}
		erpc.Infof("auth info: %s, result: %s", clientAuthInfo, ret)
		return
	},
	erpc.WithBodyCodec('s'),
)

var authChecker = auth.NewCheckerPlugin(
	func(sess auth.Session, fn auth.RecvOnce) (ret interface{}, stat *erpc.Status) {
		var authInfo string
		stat = fn(&authInfo)
		if !stat.OK() {
			return
		}
		erpc.Infof("auth info: %v", authInfo)
		if clientAuthInfo != authInfo {
			return nil, erpc.NewStatus(403, "auth fail", "auth fail detail")
		}
		return "pass", nil
	},
	erpc.WithBodyCodec('s'),
)

func TestHandshakeWebsocketAuth(t *testing.T) {
	srv := erpc.NewPeer(erpc.PeerConfig{}, handshakePlugin)
	http.Handle("/token", ws.NewJSONServeHandler(srv, nil))
	go http.ListenAndServe(":9094", nil)
	srv.RouteCall(new(P))
	time.Sleep(time.Millisecond * 200)

	// example in Browser: ws://localhost/token?access_token=clientAuthInfo
	rawQuery := fmt.Sprintf("/token?%s=%s", clientAuthKey, clientAuthInfo)
	cli := erpc.NewPeer(erpc.PeerConfig{}, ws.NewDialPlugin(rawQuery))
	sess, stat := cli.Dial(":9094", jsonSubProto.NewJSONSubProtoFunc())
	if !stat.OK() {
		t.Fatal(stat)
	}
	var result int
	stat = sess.Call("/p/divide", &Arg{
		A: 10,
		B: 2,
	}, &result,
	).Status()
	if !stat.OK() {
		t.Fatal(stat)
	}
	t.Logf("10/2=%d", result)
	time.Sleep(time.Millisecond * 200)

	// error test
	rawQuery = fmt.Sprintf("/token?%s=wrongToken", clientAuthKey)
	cli = erpc.NewPeer(erpc.PeerConfig{}, ws.NewDialPlugin(rawQuery))
	sess, stat = cli.Dial(":9094", jsonSubProto.NewJSONSubProtoFunc())
	if stat.OK() {
		t.Fatal("why dial correct by wrong token?")
	}
	time.Sleep(time.Millisecond * 200)
}

const clientAuthKey = "access_token"
const clientUserID = "user-1234"

var handshakePlugin = ws.NewHandshakeAuthPlugin(
	func(r *http.Request) (sessionId string, status *erpc.Status) {
		token := ws.QueryToken(clientAuthKey, r)
		erpc.Infof("auth token: %v", token)
		if token != clientAuthInfo {
			return "", erpc.NewStatus(erpc.CodeUnauthorized, erpc.CodeText(erpc.CodeUnauthorized))
		}
		return clientUserID, nil
	},
	func(sess erpc.Session) *erpc.Status {
		erpc.Infof("login userID: %v", sess.ID())
		return nil
	},
)
