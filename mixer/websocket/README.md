## websocket

Websocket is an extension package that makes the Teleport framework compatible with websocket protocol as specified in RFC 6455.

### Usage

`import ws "github.com/henrylee2cn/teleport/mixer/websocket"`

#### Test

```go
package websocket_test

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
	ws "github.com/henrylee2cn/teleport/mixer/websocket"
	"github.com/henrylee2cn/teleport/mixer/websocket/jsonSubProto"
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
	srv := ws.NewServer("/ws", tp.PeerConfig{ListenPort: 9090})
	srv.RouteCall(new(P))
	go srv.ListenAndServe()

	time.Sleep(time.Second * 1)

	cli := ws.NewClient(":9090", "/ws", tp.PeerConfig{})
	sess, err := cli.Dial()
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

	cli := ws.NewClient(":9091", "/ws", tp.PeerConfig{})
	cli.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
	sess, err := cli.DialProtobuf()
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
```

test command:

```sh
go test -v -run=TestJSONWebsocket
go test -v -run=TestPbWebsocketTLS
go test -v -run=TestCustomizedWebsocket
```

Among them, TestJSONWebsocket's request body is:

```json
{
  "seq": 0,
  "mtype": 1,
  "uri": "/p/divide",
  "meta": "",
  "body_codec": 106,
  "body": "{\"A\":10,\"B\":2}",
  "xfer_pipe": []
}
```

TestJSONWebsocket's response body is:

```json
{
  "seq": 0,
  "mtype": 2,
  "uri": "/p/divide",
  "meta": "",
  "body_codec": 106,
  "body": "5",
  "xfer_pipe": []
}
```