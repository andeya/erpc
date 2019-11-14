package main

import (
	"github.com/henrylee2cn/cfgo"
	"github.com/henrylee2cn/erpc/v6"
)

//go:generate go build $GOFILE

func main() {
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
