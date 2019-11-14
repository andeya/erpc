package erpcs

import (
	"testing"
	"time"

	"github.com/henrylee2cn/erpc/v6"
)

type Call struct {
	erpc.CallCtx
}

func (*Call) Test(*struct{}) (*struct{}, *erpc.Status) {
	return nil, nil
}

type Push struct {
	erpc.PushCtx
}

func (*Push) Test(*struct{}) *erpc.Status {
	return nil
}

func TestTPS(t *testing.T) {
	erpc.SetLoggerLevel("OFF")
	// Server
	srv := erpc.NewPeer(erpc.PeerConfig{ListenPort: 9090}, NewTPS(5))
	srv.RouteCall(new(Call))
	srv.RoutePush(new(Push))
	go srv.ListenAndServe()
	time.Sleep(1e9)

	// Client
	cli := erpc.NewPeer(erpc.PeerConfig{})
	sess, stat := cli.Dial(":9090")
	if !stat.OK() {
		t.Fatal(stat)
	}

	ticker := time.NewTicker(time.Millisecond * 10)
	for i := 0; i < 1<<10; i++ {
		<-ticker.C
		stat = sess.Call("/call/test", nil, nil).Status()
		if !stat.OK() {
			t.Fatal(stat)
		}
		stat = sess.Push("/push/test", nil)
		if !stat.OK() {
			t.Fatal(stat)
		}
	}
}
