package main

import (
	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/plugin"
)

func main() {
	cli := tp.NewPeer(tp.PeerConfig{}, plugin.LaunchAuth(generateAuthInfo))
	defer cli.Close()
	_, e := cli.Dial(":9090")
	if e != nil {
		tp.Fatalf("%v", e)
	}
}

const clientAuthInfo = "client auth info"

func generateAuthInfo() string {
	tp.Infof("generate auth info: %s", clientAuthInfo)
	return clientAuthInfo
}
