package main

import (
	"encoding/json"
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/socket"
)

func main() {
	go tp.GraceSignal()
	tp.SetShutdown(time.Second*20, nil, nil)
	var peer = tp.NewPeer(tp.PeerConfig{
		SlowCometDuration: time.Millisecond * 500,
		// DefaultBodyCodec:    "json",
		PrintBody:   true,
		CountTime:   true,
		RedialTimes: 3,
	})
	defer peer.Close()
	peer.PushRouter.Reg(new(Push))

	var sess, err = peer.Dial("127.0.0.1:9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}
	sess.SetId("testId")

	var reply interface{}
	for {
		var pullcmd = sess.Pull(
			"/group/home/test?peer_id=call-1",
			map[string]interface{}{
				"bytes": []byte("test bytes"),
			},
			&reply,
			socket.WithXferPipe('g'),
			socket.WithSetMeta("set", "0"),
			socket.WithAddMeta("add", "1"),
			socket.WithAddMeta("add", "2"),
		)
		if pullcmd.Rerror() != nil {
			tp.Errorf("pull error: %v", pullcmd.Rerror())
			time.Sleep(time.Second * 2)
		} else {
			break
		}
	}
	tp.Infof("test: %#v", reply)

	var pullcmd = sess.Pull(
		"/group/home/test_unknown?peer_id=call-2",
		struct {
			RawMessage json.RawMessage
			Bytes      []byte
		}{
			json.RawMessage(`{"RawMessage":"test_unknown"}`),
			[]byte("test bytes"),
		},
		&reply,
		socket.WithXferPipe('g'),
	)

	if pullcmd.Rerror() != nil {
		tp.Fatalf("pull error: %v", pullcmd.Rerror())
	}
	tp.Infof("test_unknown: %#v", reply)
}

// Push controller
type Push struct {
	tp.PushCtx
}

// Test handler
func (p *Push) Test(args *map[string]interface{}) *tp.Rerror {
	tp.Infof("receive push(%s):\nargs: %#v\nquery: %#v\n", p.Ip(), args, p.Query())
	return nil
}
