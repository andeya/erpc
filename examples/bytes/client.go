package main

import (
	"time"

	tp "github.com/henrylee2cn/teleport"
)

//go:generate go build $GOFILE

func main() {
	defer tp.FlushLogger()
	go tp.GraceSignal()
	tp.SetShutdown(time.Second*20, nil, nil)
	var peer = tp.NewPeer(tp.PeerConfig{
		SlowCometDuration: time.Millisecond * 500,
		PrintDetail:       true,
	})
	defer peer.Close()
	peer.RoutePush(new(Push))

	var sess, stat = peer.Dial("127.0.0.1:9090")
	if !stat.OK() {
		tp.Fatalf("%v", stat)
	}
	var result []byte
	for {
		if stat = sess.Call(
			"/group/home/test",
			[]byte("call text"),
			&result,
			tp.WithAddMeta("peer_id", "call-1"),
		).Status(); !stat.OK() {
			tp.Errorf("call error: %v", stat)
			time.Sleep(time.Second * 2)
		} else {
			break
		}
	}
	tp.Infof("test result: %s", result)

	stat = sess.Call(
		"/group/home/test_unknown",
		[]byte("unknown call text"),
		&result,
		tp.WithAddMeta("peer_id", "call-2"),
	).Status()
	if tp.IsConnError(stat) {
		tp.Fatalf("has conn error: %v", stat)
	}
	if !stat.OK() {
		tp.Fatalf("call error: %v", stat)
	}
	tp.Infof("test_unknown: %s", result)
}

// Push controller
type Push struct {
	tp.PushCtx
}

// Test handler
func (p *Push) Test(arg *[]byte) *tp.Status {
	tp.Infof("receive push(%s):\narg: %s\n", p.IP(), *arg)
	return nil
}
