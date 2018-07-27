package main

import (
	"encoding/json"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	go tp.GraceSignal()
	tp.SetShutdown(time.Second*20, nil, nil)
	var peer = tp.NewPeer(tp.PeerConfig{
		SlowCometDuration: time.Millisecond * 500,
		// DefaultBodyCodec:    "json",
		// DefaultContextAge: time.Second * 5,
		PrintDetail: true,
		CountTime:   true,
		RedialTimes: 3,
	})
	defer peer.Close()
	peer.RoutePush(new(Push))

	var sess, rerr = peer.Dial("127.0.0.1:9090")
	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
	sess.SetId("testId")

	var result interface{}
	for {
		if rerr = sess.Call(
			"/group/home/test?peer_id=call-1",
			map[string]interface{}{
				"bytes": []byte("test bytes"),
			},
			&result,
			tp.WithBodyCodec('j'),
			tp.WithAcceptBodyCodec('j'),
			tp.WithXferPipe('g'),
			tp.WithSetMeta("set", "0"),
			tp.WithAddMeta("add", "1"),
			tp.WithAddMeta("add", "2"),
		).Rerror(); rerr != nil {
			tp.Errorf("call error: %v", rerr)
			time.Sleep(time.Second * 2)
		} else {
			break
		}
	}
	tp.Infof("test: %#v", result)

	// sess.Close()

	rerr = sess.Call(
		"/group/home/test_unknown?peer_id=call-2",
		struct {
			RawMessage json.RawMessage
			Bytes      []byte
		}{
			json.RawMessage(`{"RawMessage":"test_unknown"}`),
			[]byte("test bytes"),
		},
		&result,
		tp.WithXferPipe('g'),
	).Rerror()
	if tp.IsConnRerror(rerr) {
		tp.Fatalf("has conn rerror: %v", rerr)
	}
	if rerr != nil {
		tp.Fatalf("call error: %v", rerr)
	}
	tp.Infof("test_unknown: %#v", result)
}

// Push controller
type Push struct {
	tp.PushCtx
}

// Test handler
func (p *Push) Test(arg *map[string]interface{}) *tp.Rerror {
	tp.Infof("receive push(%s):\narg: %#v\nquery: %#v\n", p.Ip(), arg, p.Query())
	return nil
}
