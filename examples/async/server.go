package main

import (
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	svr := tp.NewPeer(tp.PeerConfig{
		CountTime:     true,
		ListenAddress: ":9090",
	})
	svr.RoutePull(new(test))
	svr.Listen()
}

type test struct {
	tp.PullCtx
}

func (t *test) Wait3s(args *string) (string, *tp.Rerror) {
	time.Sleep(3 * time.Second)
	return *args + " -> OK", nil
}
