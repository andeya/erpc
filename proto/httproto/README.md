## httproto

httproto is implemented HTTP style socket communication protocol.

NOTE: It simply transfers data in HTTP style instead of the full HTTP protocol.

### Message Bytes

example:

- Request Message

```
POST /home/test?peer_id=110 HTTP/1.1
Host: localhost:9090
User-Agent: Go-http-client/1.1
Content-Length: 24
Content-Type: application/json;charset=utf-8
Accept-Encoding: gzip
X-Mtype: 1
X-Seq: 1

{"author":"henrylee2cn"}
```

- Response Message

```
HTTP/1.1 200 OK
Content-Length: 32
Content-Type: application/json
X-Mtype: 2
X-Seq: 1

{"arg":{"author":"henrylee2cn"}}
```

### Usage

`import "github.com/henrylee2cn/teleport/proto/httproto"`

#### Test

```go
package httproto_test

import (
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/proto/httproto"
	"github.com/henrylee2cn/teleport/xfer/gzip"
)

type Home struct {
	tp.CallCtx
}

func (h *Home) Test(arg *map[string]string) (map[string]interface{}, *tp.Rerror) {
	tp.Infof("peer_id: %s", h.PeekMeta("peer_id"))
	return map[string]interface{}{
		"arg": *arg,
	}, nil
}

func TestHTTProto(t *testing.T) {
	gzip.Reg('g', "gizp-5", 5)

	// Server
	srv := tp.NewPeer(tp.PeerConfig{ListenPort: 9090})
	srv.RouteCall(new(Home))
	go srv.ListenAndServe(httproto.NewHTTProtoFunc())
	time.Sleep(1e9)

	// Client
	cli := tp.NewPeer(tp.PeerConfig{})
	sess, err := cli.Dial(":9090", httproto.NewHTTProtoFunc())
	if err != nil {
		t.Error(err)
	}
	var result interface{}
	rerr := sess.Call("/home/test?peer_id=110",
		map[string]string{
			"author": "henrylee2cn",
		},
		&result,
		tp.WithXferPipe('g'),
	).Rerror()
	if rerr != nil {
		t.Error(rerr)
	}
	t.Logf("result:%v", result)
	time.Sleep(3e9)
}
```

test command:

```sh
go test -v -run=TestHTTProto
```
