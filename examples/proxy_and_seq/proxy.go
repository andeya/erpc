package main

import (
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/plugin/proxy"
)

func main() {
	srv := tp.NewPeer(
		tp.PeerConfig{
			ListenPort: 8080,
		},
		newProxyPlugin(),
	)
	srv.ListenAndServe()
}

func newProxyPlugin() tp.Plugin {
	cli := tp.NewPeer(tp.PeerConfig{RedialTimes: 3})
	var sess tp.Session
	var rerr *tp.Rerror
DIAL:
	sess, rerr = cli.Dial(":9090")
	if rerr != nil {
		tp.Warnf("%v", rerr)
		time.Sleep(time.Second * 3)
		goto DIAL
	}
	return proxy.Proxy(func(*proxy.ProxyLabel) proxy.Forwarder {
		return sess
	})
}
