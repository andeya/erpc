package main

import (
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	srv := tp.NewPeer(tp.PeerConfig{
		CountTime:     true,
		ListenAddress: ":9090",
	})
	srv.RoutePull(new(test))
	srv.ListenAndServe()
}

type test struct {
	tp.PullCtx
}

func (t *test) Wait3s(args *string) (string, *tp.Rerror) {
	time.Sleep(3 * time.Second)
	return *args + " -> OK", nil
}
