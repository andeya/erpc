package config

import (
	"testing"

	"github.com/andeya/cfgo"
	"github.com/andeya/erpc/v7"
	"github.com/andeya/goutil"
)

//go:generate go test -v -c -o "${GOPACKAGE}_client" $GOFILE

func TestClient(t *testing.T) {
	if goutil.IsGoTest() {
		t.Log("skip test in go test")
		return
	}
	defer erpc.FlushLogger()
	cfg := erpc.PeerConfig{}

	// auto create and sync config/config.yaml
	cfgo.MustGet("config/config.yaml", true).MustReg("cfg_cli", &cfg)

	cli := erpc.NewPeer(cfg)
	defer cli.Close()

	sess, stat := cli.Dial(":9090")
	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}

	var result int
	stat = sess.Call("/math/add",
		[]int{1, 2, 3, 4, 5},
		&result,
	).Status()

	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}
	erpc.Printf("result: 1+2+3+4+5 = %d", result)
}
