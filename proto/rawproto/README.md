## rawproto

rawproto is the protocol default frame splicing format protocol.

### Message Bytes

raw protocol format(Big Endian):

```sh
{4 bytes message length}
{1 byte protocol version} # 6
{1 byte transfer pipe length}
{transfer pipe IDs}
# The following is handled data by transfer pipe
{1 bytes sequence length}
{sequence (HEX 36 string of int32)}
{1 byte message type} # e.g. CALL:1; REPLY:2; PUSH:3
{1 bytes service method length}
{service method}
{2 bytes status length}
{status(urlencoded)}
{2 bytes metadata length}
{metadata(urlencoded)}
{1 byte body codec id}
{body}
```

### Usage

`import "github.com/henrylee2cn/erpc/v6/proto/pbproto"`

#### Test

```go
package rawproto_test

import (
	"testing"
	"time"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/xfer/gzip"
)

func TestRawProto(t *testing.T) {
	gzip.Reg('g', "gizp-5", 5)

	// server
	srv := erpc.NewPeer(erpc.PeerConfig{ListenPort: 9090})
	srv.RouteCall(new(Home))
	go srv.ListenAndServe()
	time.Sleep(1e9)

	// client
	cli := erpc.NewPeer(erpc.PeerConfig{})
	cli.RoutePush(new(Push))
	sess, stat := cli.Dial(":9090")
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
go test -v -run=TestRawProto
```
