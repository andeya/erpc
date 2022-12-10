package tps

import (
	"testing"
	"time"

	"github.com/andeya/erpc/v7"
	"github.com/andeya/goutil"
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

//go:generate go test -v -c -o "${GOPACKAGE}" ./...

func TestTPS(t *testing.T) {
	if goutil.IsGoTest() {
		t.Log("skip test in go test")
		return
	}
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
