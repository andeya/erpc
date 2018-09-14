## rawproto

rawproto is the protocol default frame splicing format protocol.

### Message Bytes

```sh
{4 bytes message length}
{1 byte protocol version}
{1 byte transfer pipe length}
{transfer pipe IDs}
# The following is handled data by transfer pipe
{2 bytes sequence length}
{sequence}
{1 byte message type} # e.g. CALL:1; REPLY:2; PUSH:3
{2 bytes URI length}
{URI}
{2 bytes metadata length}
{metadata(urlencoded)}
{1 byte body codec id}
{body}
```

NOTE: Big Endian

### Usage

`import "github.com/henrylee2cn/teleport/proto/pbproto"`

#### Test

```go
package pbproto_test

import (
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/proto/pbproto"
	"github.com/henrylee2cn/teleport/xfer/gzip"
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

func TestPbProto(t *testing.T) {
	gzip.Reg('g', "gizp-5", 5)

	// server
	srv := tp.NewPeer(tp.PeerConfig{ListenPort: 9090})
	srv.RouteCall(new(Home))
	go srv.ListenAndServe(pbproto.NewPbProtoFunc)
	time.Sleep(1e9)

	// client
	cli := tp.NewPeer(tp.PeerConfig{})
	cli.RoutePush(new(Push))
	sess, err := cli.Dial(":9090", pbproto.NewPbProtoFunc)
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
go test -v -run=TestPbProto
```
