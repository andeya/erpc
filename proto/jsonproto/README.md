## jsonproto

jsonproto is implemented JSON socket communication protocol.


### Data Packet

`{length bytes}` `{xfer_pipe length byte}` `{xfer_pipe bytes}` `{JSON bytes}`

- `{length bytes}`: uint32, 4 bytes, big endian
- `{xfer_pipe length byte}`: 1 byte
- `{xfer_pipe bytes}`: one byte one xfer
- `{JSON bytes}`: {"seq":%d,"ptype":%d,"uri":%q,"meta":%q,"body_codec":%d,"body":"%s"}

### Usage

`import "github.com/henrylee2cn/teleport/proto/jsonproto"`

#### Test

```go
package jsonproto_test

import (
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/proto/jsonproto"
)

type Home struct {
	tp.CallCtx
}

func (h *Home) Test(arg *map[string]interface{}) (map[string]interface{}, *tp.Rerror) {
	h.Session().Push("/push/test", map[string]interface{}{
		"your_id": h.Query().Get("peer_id"),
	})
	meta := h.CopyMeta()
	return map[string]interface{}{
		"arg":  *arg,
		"meta": meta.String(),
	}, nil
}

func TestJsonProto(t *testing.T) {
	gzip.Reg('g', "gizp-5", 5)
	
	// Server
	srv := tp.NewPeer(tp.PeerConfig{ListenPort: 9090})
	srv.RouteCall(new(Home))
	go srv.ListenAndServe(jsonproto.NewJsonProtoFunc)
	time.Sleep(1e9)

	// Client
	cli := tp.NewPeer(tp.PeerConfig{})
	cli.RoutePush(new(Push))
	sess, err := cli.Dial(":9090", jsonproto.NewJsonProtoFunc)
	if err != nil {
		t.Error(err)
	}
	var result interface{}
	rerr := sess.Call("/home/test?peer_id=110",
		map[string]interface{}{
			"bytes": []byte("test bytes"),
		},
		&result,
		tp.WithAddMeta("add", "1"),
		tp.WithXferPipe('g'),
	).Rerror()
	if rerr != nil {
		t.Error(rerr)
	}
	t.Logf("result:%v", result)
	time.Sleep(3e9)
}

type Push struct {
	tp.PushCtx
}

func (p *Push) Test(arg *map[string]interface{}) *tp.Rerror {
	tp.Infof("receive push(%s):\narg: %#v\n", p.Ip(), arg)
	return nil
}
```

test command:

```sh
go test -v -run=TestJsonProto
```
