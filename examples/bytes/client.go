package main

import (
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	go tp.GraceSignal()
	tp.SetShutdown(time.Second*20, nil, nil)
	var peer = tp.NewPeer(tp.PeerConfig{
		SlowCometDuration: time.Millisecond * 500,
		PrintDetail:       true,
	})
	defer peer.Close()
	peer.RoutePush(new(Push))

	var sess, rerr = peer.Dial("127.0.0.1:9090")
	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
	var result []byte
	for {
		if rerr = sess.Call(
			"/group/home/test?peer_id=call-1",
			[]byte("call text"),
			&result,
			tp.WithQuery("a", "1"),
		).Rerror(); rerr != nil {
			tp.Errorf("call error: %v", rerr)
			time.Sleep(time.Second * 2)
		} else {
			break
		}
	}
	tp.Infof("test result: %s", result)

	rerr = sess.Call(
		"/group/home/test_unknown?peer_id=call-2",
		[]byte("unknown call text"),
		&result,
		tp.WithQuery("b", "2"),
	).Rerror()
	if tp.IsConnRerror(rerr) {
		tp.Fatalf("has conn rerror: %v", rerr)
	}
	if rerr != nil {
		tp.Fatalf("call error: %v", rerr)
	}
	tp.Infof("test_unknown: %s", result)
}

// Push controller
type Push struct {
	tp.PushCtx
}

// Test handler
func (p *Push) Test(arg *[]byte) *tp.Rerror {
	tp.Infof("receive push(%s):\narg: %s\nquery: %#v\n", p.Ip(), *arg, p.Query())
	return nil
}
