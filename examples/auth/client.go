package main

import (
	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/plugin/auth"
)

//go:generate go build $GOFILE

func main() {
	defer tp.FlushLogger()
	cli := tp.NewPeer(tp.PeerConfig{}, auth.NewBearerPlugin(authBearer))
	defer cli.Close()
	_, rerr := cli.Dial(":9090")
	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
}

const clientAuthInfo = "client-auth-info-12345"

func authBearer(sess auth.Session, fn auth.Sender) *tp.Rerror {
	var ret string
	rerr := fn(clientAuthInfo, &ret)
	if rerr.HasError() {
		tp.Infof("auth info: %s, error: %s", clientAuthInfo, rerr)
		return rerr
	}
	tp.Infof("auth info: %s, result: %s", clientAuthInfo, ret)
	return nil
}
