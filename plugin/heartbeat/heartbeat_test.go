package heartbeat_test

import (
	"testing"
	"time"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/plugin/heartbeat"
)

func TestHeartbeatCall1(t *testing.T) {
	srv := erpc.NewPeer(
		erpc.PeerConfig{ListenPort: 9090, PrintDetail: true},
		heartbeat.NewPong(),
	)
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := erpc.NewPeer(
		erpc.PeerConfig{PrintDetail: true},
		heartbeat.NewPing(3, true),
	)
	cli.Dial(":9090")
	time.Sleep(time.Second * 10)
}

func TestHeartbeatCall2(t *testing.T) {
	srv := erpc.NewPeer(
		erpc.PeerConfig{ListenPort: 9090, PrintDetail: true},
		heartbeat.NewPong(),
	)
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := erpc.NewPeer(
		erpc.PeerConfig{PrintDetail: true},
		heartbeat.NewPing(3, true),
	)
	sess, _ := cli.Dial(":9090")
	for i := 0; i < 8; i++ {
		sess.Call("/", nil, nil)
		time.Sleep(time.Second)
	}
	time.Sleep(time.Second * 5)
}

func TestHeartbeatPush1(t *testing.T) {
	srv := erpc.NewPeer(
		erpc.PeerConfig{ListenPort: 9090, PrintDetail: true},
		heartbeat.NewPing(3, false),
	)
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := erpc.NewPeer(
		erpc.PeerConfig{PrintDetail: true},
		heartbeat.NewPong(),
	)
	cli.Dial(":9090")
	time.Sleep(time.Second * 10)
}

func TestHeartbeatPush2(t *testing.T) {
	srv := erpc.NewPeer(
		erpc.PeerConfig{ListenPort: 9090, PrintDetail: true},
		heartbeat.NewPing(3, false),
	)
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := erpc.NewPeer(
		erpc.PeerConfig{PrintDetail: true},
		heartbeat.NewPong(),
	)
	sess, _ := cli.Dial(":9090")
	for i := 0; i < 8; i++ {
		sess.Push("/", nil)
		time.Sleep(time.Second)
	}
	time.Sleep(time.Second * 5)
}
