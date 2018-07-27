package heartbeat_test

import (
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/plugin/heartbeat"
)

func TestHeartbeatCall1(t *testing.T) {
	srv := tp.NewPeer(
		tp.PeerConfig{ListenPort: 9090},
		heartbeat.NewPong(),
	)
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := tp.NewPeer(
		tp.PeerConfig{},
		heartbeat.NewPing(3, true),
	)
	cli.Dial(":9090")
	time.Sleep(time.Second * 10)
}

func TestHeartbeatCall2(t *testing.T) {
	srv := tp.NewPeer(
		tp.PeerConfig{ListenPort: 9090},
		heartbeat.NewPong(),
	)
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := tp.NewPeer(
		tp.PeerConfig{},
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
	srv := tp.NewPeer(
		tp.PeerConfig{ListenPort: 9090},
		heartbeat.NewPing(3, false),
	)
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := tp.NewPeer(
		tp.PeerConfig{},
		heartbeat.NewPong(),
	)
	cli.Dial(":9090")
	time.Sleep(time.Second * 10)
}

func TestHeartbeatPush2(t *testing.T) {
	srv := tp.NewPeer(
		tp.PeerConfig{ListenPort: 9090},
		heartbeat.NewPing(3, false),
	)
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := tp.NewPeer(
		tp.PeerConfig{},
		heartbeat.NewPong(),
	)
	sess, _ := cli.Dial(":9090")
	for i := 0; i < 8; i++ {
		sess.Push("/", nil)
		time.Sleep(time.Second)
	}
	time.Sleep(time.Second * 5)
}
