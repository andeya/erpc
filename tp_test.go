package tp_test

import (
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func panic_call(tp.CallCtx, *interface{}) (interface{}, *tp.Rerror) {
	panic("panic_call")
}

func panic_push(tp.PushCtx, *interface{}) *tp.Rerror {
	panic("panic_push")
}

func TestPanic(t *testing.T) {
	srv := tp.NewPeer(tp.PeerConfig{
		CountTime:  true,
		ListenPort: 9090,
	})
	srv.RouteCallFunc(panic_call)
	srv.RoutePushFunc(panic_push)
	go srv.ListenAndServe()

	time.Sleep(2 * time.Second)

	cli := tp.NewPeer(tp.PeerConfig{})
	defer cli.Close()
	sess, err := cli.Dial(":9090")
	if err != nil {
		t.Fatalf("%v", err)
	}
	rerr := sess.Call("/panic/call", nil, nil).Rerror()
	if rerr == nil {
		t.Fatalf("/panic/call: expect error!")
	}
	t.Logf("/panic/call error: %v", rerr)
	rerr = sess.Push("/panic/push", nil)
	if rerr != nil {
		t.Fatalf("/panic/push: expect ok!")
	}
	t.Logf("/panic/push: ok")
}
