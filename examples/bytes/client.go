package main

import (
	"time"

	"github.com/henrylee2cn/erpc/v6"
)

//go:generate go build $GOFILE

func main() {
	defer erpc.FlushLogger()
	go erpc.GraceSignal()
	erpc.SetShutdown(time.Second*20, nil, nil)
	var peer = erpc.NewPeer(erpc.PeerConfig{
		SlowCometDuration: time.Millisecond * 500,
		PrintDetail:       true,
	})
	defer peer.Close()
	peer.RoutePush(new(Push))

	var sess, stat = peer.Dial("127.0.0.1:9090")
	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}
	var result []byte
	for {
		if stat = sess.Call(
			"/group/home/test",
			[]byte("call text"),
			&result,
			erpc.WithAddMeta("peer_id", "call-1"),
		).Status(); !stat.OK() {
			erpc.Errorf("call error: %v", stat)
			time.Sleep(time.Second * 2)
		} else {
			break
		}
	}
	erpc.Infof("test result: %s", result)

	stat = sess.Call(
		"/group/home/test_unknown",
		[]byte("unknown call text"),
		&result,
		erpc.WithAddMeta("peer_id", "call-2"),
	).Status()
	if erpc.IsConnError(stat) {
		erpc.Fatalf("has conn error: %v", stat)
	}
	if !stat.OK() {
		erpc.Fatalf("call error: %v", stat)
	}
	erpc.Infof("test_unknown: %s", result)
}

// Push controller
type Push struct {
	erpc.PushCtx
}

// Test handler
func (p *Push) Test(arg *[]byte) *erpc.Status {
	erpc.Infof("receive push(%s):\narg: %s\n", p.IP(), *arg)
	return nil
}
