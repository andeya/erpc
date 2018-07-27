## websocket

Websocket is an extension package that makes the Teleport framework compatible with websocket protocol as specified in RFC 6455.

### Usage

`import ws "github.com/henrylee2cn/teleport/mixer/websocket"`

#### Test

```go
package websocket_test

import (
	"net/http"
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
	ws "github.com/henrylee2cn/teleport/mixer/websocket"
	"github.com/henrylee2cn/teleport/mixer/websocket/jsonSubProto"
	"github.com/henrylee2cn/teleport/mixer/websocket/pbSubProto"
)

type Arg struct {
	A int
	B int `param:"<range:1:>"`
}

type P struct{ tp.CallCtx }

func (p *P) Divide(arg *Arg) (int, *tp.Rerror) {
	return arg.A / arg.B, nil
}

func TestJsonSubWebsocket(t *testing.T) {
	srv := tp.NewPeer(tp.PeerConfig{})
	http.Handle("/ws", ws.NewJsonServeHandler(srv, nil))
	go http.ListenAndServe("0.0.0.0:9090", nil)
	srv.RouteCall(new(P))
	time.Sleep(time.Second * 1)

	cli := tp.NewPeer(tp.PeerConfig{}, ws.NewDialPlugin("/ws"))
	sess, err := cli.Dial("127.0.0.1:9090", jsonSubProto.NewJsonSubProtoFunc)
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

func TestPbSubWebsocket(t *testing.T) {
	srv := tp.NewPeer(tp.PeerConfig{})
	http.Handle("/ws", ws.NewPbServeHandler(srv, nil))
	go http.ListenAndServe("0.0.0.0:9090", nil)
	srv.RouteCall(new(P))
	time.Sleep(time.Second * 1)

	cli := tp.NewPeer(tp.PeerConfig{}, ws.NewDialPlugin("/ws"))
	sess, err := cli.Dial("127.0.0.1:9090", pbSubProto.NewPbSubProtoFunc)
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
go test -v -run=TestJsonSubWebsocket
go test -v -run=TestPbSubWebsocket
```

Among them, TestJsonSubWebsocket's request body is:

```json
{
  "seq": 0,
  "ptype": 1,
  "uri": "/p/divide",
  "meta": "",
  "body_codec": 106,
  "body": "{\"A\":10,\"B\":2}",
  "xfer_pipe": []
}
```

TestJsonSubWebsocket's response body is:

```json
{
  "seq": 0,
  "ptype": 2,
  "uri": "/p/divide",
  "meta": "",
  "body_codec": 106,
  "body": "5",
  "xfer_pipe": []
}
```