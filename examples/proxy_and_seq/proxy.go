package main

import (
	"time"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/plugin/proxy"
)

//go:generate go build $GOFILE

func main() {
	defer erpc.FlushLogger()
	srv := erpc.NewPeer(
		erpc.PeerConfig{
			ListenPort: 8080,
		},
		newProxyPlugin(),
	)
	srv.ListenAndServe()
}

func newProxyPlugin() erpc.Plugin {
	cli := erpc.NewPeer(erpc.PeerConfig{RedialTimes: 3})
	var sess erpc.Session
	var stat *erpc.Status
DIAL:
	sess, stat = cli.Dial(":9090")
	if !stat.OK() {
		erpc.Warnf("%v", stat)
		time.Sleep(time.Second * 3)
		goto DIAL
	}
	return proxy.NewPlugin(func(*proxy.Label) proxy.Forwarder {
		return sess
	})
}
