## jsonproto

jsonproto is implemented JSON socket communication protocol.


### Message Bytes

`{length bytes}` `{xferPipe length byte}` `{xferPipe bytes}` `{JSON bytes}`

- `{length bytes}`: uint32, 4 bytes, big endian
- `{xferPipe length byte}`: 1 byte
- `{xferPipe bytes}`: one byte one xfer
- `{JSON bytes}`: {"seq":%d,"mtype":%d,"serviceMethod":%q,"status":%q,"meta":%q,"bodyCodec":%d,"body":"%s"}

### Usage

`import "github.com/henrylee2cn/erpc/v6/proto/jsonproto"`

#### Test

```go
package jsonproto_test

import (
	"testing"
	"time"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/proto/jsonproto"
	"github.com/henrylee2cn/erpc/v6/xfer/gzip"
)

type Home struct {
	erpc.CallCtx
}

func (h *Home) Test(arg *map[string]string) (map[string]interface{}, *erpc.Status) {
	h.Session().Push("/push/test", map[string]string{
		"your_id": string(h.PeekMeta("peer_id")),
	})
	return map[string]interface{}{
		"arg": *arg,
	}, nil
}

func TestJSONProto(t *testing.T) {
	gzip.Reg('g', "gizp-5", 5)

	// Server
	srv := erpc.NewPeer(erpc.PeerConfig{ListenPort: 9090})
	srv.RouteCall(new(Home))
	go srv.ListenAndServe(jsonproto.NewJSONProtoFunc())
	time.Sleep(1e9)

	// Client
	cli := erpc.NewPeer(erpc.PeerConfig{})
	cli.RoutePush(new(Push))
	sess, stat := cli.Dial(":9090", jsonproto.NewJSONProtoFunc())
	if !stat.OK() {
		t.Fatal(stat)
	}
	var result interface{}
	stat = sess.Call("/home/test",
		map[string]string{
			"author": "henrylee2cn",
		},
		&result,
		erpc.WithAddMeta("peer_id", "110"),
		erpc.WithXferPipe('g'),
	).Status()
	if !stat.OK() {
		t.Error(stat)
	}
	t.Logf("result:%v", result)
	time.Sleep(3e9)
}

type Push struct {
	erpc.PushCtx
}

func (p *Push) Test(arg *map[string]string) *erpc.Status {
	erpc.Infof("receive push(%s):\narg: %#v\n", p.IP(), arg)
	return nil
}
```

test command:

```sh
go test -v -run=TestJSONProto
```
