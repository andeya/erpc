package websocket_test

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
	ws "github.com/henrylee2cn/teleport/mixer/websocket"
	"github.com/henrylee2cn/teleport/mixer/websocket/jsonSubProto"
	"github.com/henrylee2cn/teleport/plugin/auth"
)

type Arg struct {
	A int
	B int `param:"<range:1:>"`
}

type P struct{ tp.CallCtx }

func (p *P) Divide(arg *Arg) (int, *tp.Rerror) {
	return arg.A / arg.B, nil
}

func TestJSONWebsocket(t *testing.T) {
	srv := ws.NewServer("/", tp.PeerConfig{ListenPort: 9090})
	srv.RouteCall(new(P))
	go srv.ListenAndServe()

	time.Sleep(time.Second * 1)

	cli := ws.NewClient("/", tp.PeerConfig{})
	sess, err := cli.Dial(":9090")
	if err != nil {
		t.Fatal(err)
	}
	var result int
	rerr := sess.Call("/p/divide", &Arg{
		A: 10,
		B: 2,
	}, &result,
	).Rerror()
	if rerr != nil {
		t.Fatal(rerr)
	}
	t.Logf("10/2=%d", result)
	time.Sleep(time.Second)
}

func TestPbWebsocketTLS(t *testing.T) {
	srv := ws.NewServer("/ws", tp.PeerConfig{ListenPort: 9091})
	srv.RouteCall(new(P))
	srv.SetTLSConfig(tp.GenerateTLSConfigForServer())
	go srv.ListenAndServeProtobuf()

	time.Sleep(time.Second * 1)

	cli := ws.NewClient("/ws", tp.PeerConfig{})
	cli.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
	sess, err := cli.DialProtobuf(":9091")
	if err != nil {
		t.Fatal(err)
	}
	var result int
	rerr := sess.Call("/p/divide", &Arg{
		A: 10,
		B: 2,
	}, &result,
	).Rerror()
	if rerr != nil {
		t.Fatal(rerr)
	}
	t.Logf("10/2=%d", result)
	time.Sleep(time.Second)
}

func TestCustomizedWebsocket(t *testing.T) {
	srv := tp.NewPeer(tp.PeerConfig{})
	http.Handle("/ws", ws.NewJSONServeHandler(srv, nil))
	go http.ListenAndServe(":9092", nil)
	srv.RouteCall(new(P))
	time.Sleep(time.Second * 1)

	cli := tp.NewPeer(tp.PeerConfig{}, ws.NewDialPlugin("/ws"))
	sess, err := cli.Dial(":9092", jsonSubProto.NewJSONSubProtoFunc())
	if err != nil {
		t.Fatal(err)
	}
	var result int
	rerr := sess.Call("/p/divide", &Arg{
		A: 10,
		B: 2,
	}, &result,
	).Rerror()
	if rerr != nil {
		t.Fatal(rerr)
	}
	t.Logf("10/2=%d", result)
	time.Sleep(time.Second)
}

func TestJSONWebsocketAuth(t *testing.T) {
	srv := ws.NewServer(
		"/",
		tp.PeerConfig{ListenPort: 9090},
		authChecker,
	)
	srv.RouteCall(new(P))
	go srv.ListenAndServe()

	time.Sleep(time.Second * 1)

	cli := ws.NewClient(
		"/",
		tp.PeerConfig{},
		authBearer,
	)
	sess, err := cli.Dial(":9090")
	if err != nil {
		t.Fatal(err)
	}
	var result int
	rerr := sess.Call("/p/divide", &Arg{
		A: 10,
		B: 2,
	}, &result,
	).Rerror()
	if rerr != nil {
		t.Fatal(rerr)
	}
	t.Logf("10/2=%d", result)
	time.Sleep(time.Second)
}

const clientAuthInfo = "client-auth-info-12345"

var authBearer = auth.NewBearerPlugin(
	func(sess auth.Session, fn auth.SendOnce) (rerr *tp.Rerror) {
		var ret string
		rerr = fn(clientAuthInfo, &ret)
		if rerr.HasError() {
			return
		}
		tp.Infof("auth info: %s, result: %s", clientAuthInfo, ret)
		return
	},
	tp.WithBodyCodec('s'),
)

var authChecker = auth.NewCheckerPlugin(
	func(sess auth.Session, fn auth.RecvOnce) (ret interface{}, rerr *tp.Rerror) {
		var authInfo string
		rerr = fn(&authInfo)
		if rerr.HasError() {
			return
		}
		tp.Infof("auth info: %v", authInfo)
		if clientAuthInfo != authInfo {
			return nil, tp.NewRerror(403, "auth fail", "auth fail detail")
		}
		return "pass", nil
	},
	tp.WithBodyCodec('s'),
)
