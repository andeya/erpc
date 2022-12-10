package heartbeat_test

import (
	"testing"
	"time"

	"github.com/andeya/erpc/v7"
	"github.com/andeya/erpc/v7/plugin/heartbeat"
	"github.com/andeya/goutil"
)

//go:generate go test -v -c -o "${GOPACKAGE}" $GOFILE

func TestHeartbeatCall1(t *testing.T) {
	if goutil.IsGoTest() {
		t.Log("skip test in go test")
		return
	}
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
	if goutil.IsGoTest() {
		t.Log("skip test in go test")
		return
	}
	srv := erpc.NewPeer(
		erpc.PeerConfig{ListenPort: 9091, PrintDetail: true},
		heartbeat.NewPong(),
	)
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := erpc.NewPeer(
		erpc.PeerConfig{PrintDetail: true},
		heartbeat.NewPing(3, true),
	)
	sess, _ := cli.Dial(":9091")
	for i := 0; i < 8; i++ {
		sess.Call("/", nil, nil)
		time.Sleep(time.Second)
	}
	time.Sleep(time.Second * 5)
}

func TestHeartbeatPush1(t *testing.T) {
	if goutil.IsGoTest() {
		t.Log("skip test in go test")
		return
	}
	srv := erpc.NewPeer(
		erpc.PeerConfig{ListenPort: 9092, PrintDetail: true},
		heartbeat.NewPing(3, false),
	)
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := erpc.NewPeer(
		erpc.PeerConfig{PrintDetail: true},
		heartbeat.NewPong(),
	)
	cli.Dial(":9092")
	time.Sleep(time.Second * 10)
}

func TestHeartbeatPush2(t *testing.T) {
	if goutil.IsGoTest() {
		t.Log("skip test in go test")
		return
	}
	srv := erpc.NewPeer(
		erpc.PeerConfig{ListenPort: 9093, PrintDetail: true},
		heartbeat.NewPing(3, false),
	)
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := erpc.NewPeer(
		erpc.PeerConfig{PrintDetail: true},
		heartbeat.NewPong(),
	)
	sess, _ := cli.Dial(":9093")
	for i := 0; i < 8; i++ {
		sess.Push("/", nil)
		time.Sleep(time.Second)
	}
	time.Sleep(time.Second * 5)
}
