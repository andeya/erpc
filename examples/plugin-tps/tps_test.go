package tps

import (
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

type Call struct {
	tp.CallCtx
}

func (*Call) Test(*struct{}) (*struct{}, *tp.Status) {
	return nil, nil
}

type Push struct {
	tp.PushCtx
}

func (*Push) Test(*struct{}) *tp.Status {
	return nil
}

func TestTPS(t *testing.T) {
	tp.SetLoggerLevel("OFF")
	// Server
	srv := tp.NewPeer(tp.PeerConfig{ListenPort: 9090}, NewTPS(5))
	srv.RouteCall(new(Call))
	srv.RoutePush(new(Push))
	go srv.ListenAndServe()
	time.Sleep(1e9)

	// Client
	cli := tp.NewPeer(tp.PeerConfig{})
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
