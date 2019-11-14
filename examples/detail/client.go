package main

import (
	"encoding/json"
	"time"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/xfer/gzip"
)

//go:generate go build $GOFILE

func main() {
	defer erpc.FlushLogger()
	gzip.Reg('g', "gizp", 5)

	go erpc.GraceSignal()
	erpc.SetShutdown(time.Second*20, nil, nil)
	var peer = erpc.NewPeer(erpc.PeerConfig{
		SlowCometDuration: time.Millisecond * 500,
		// DefaultBodyCodec:    "json",
		// DefaultContextAge: time.Second * 5,
		PrintDetail:    true,
		CountTime:      true,
		RedialTimes:    3,
		RedialInterval: time.Second * 3,
	})
	defer peer.Close()
	peer.RoutePush(new(Push))

	var sess, stat = peer.Dial("127.0.0.1:9090")
	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}
	sess.SetID("testId")

	var result interface{}
	for {
		if stat = sess.Call(
			"/group/home/test",
			map[string]interface{}{
				"bytes": []byte("test bytes"),
			},
			&result,
			erpc.WithBodyCodec('j'),
			erpc.WithAcceptBodyCodec('j'),
			erpc.WithXferPipe('g'),
			erpc.WithSetMeta("peer_id", "call-1"),
			erpc.WithAddMeta("add", "1"),
			erpc.WithAddMeta("add", "2"),
		).Status(); !stat.OK() {
			erpc.Errorf("call error: %v", stat)
			time.Sleep(time.Second * 2)
		} else {
			break
		}
	}
	erpc.Infof("test: %#v", result)

	// sess.Close()

	stat = sess.Call(
		"/group/home/test_unknown",
		struct {
			RawMessage json.RawMessage
			Bytes      []byte
		}{
			json.RawMessage(`{"RawMessage":"test_unknown"}`),
			[]byte("test bytes"),
		},
		&result,
		erpc.WithXferPipe('g'),
	).Status()
	if erpc.IsConnError(stat) {
		erpc.Fatalf("has conn error: %v", stat)
	}
	if !stat.OK() {
		erpc.Fatalf("call error: %v", stat)
	}
	erpc.Infof("test_unknown: %#v", result)
}

// Push controller
type Push struct {
	erpc.PushCtx
}

// Test handler
func (p *Push) Test(arg *map[string]interface{}) *erpc.Status {
	erpc.Infof("receive push(%s):\narg: %#v\n", p.IP(), arg)
	return nil
}
