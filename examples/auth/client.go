package main

import (
	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/plugin/auth"
)

func main() {
	cli := tp.NewPeer(tp.PeerConfig{}, auth.LaunchAuth(generateAuthInfo))
	defer cli.Close()
	_, rerr := cli.Dial(":9090")
	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
}

const clientAuthInfo = "client-auth-info-12345"

func generateAuthInfo() string {
	tp.Infof("generate auth info: %s", clientAuthInfo)
	return clientAuthInfo
}
