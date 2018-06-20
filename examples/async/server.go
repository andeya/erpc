package main

import (
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	srv := tp.NewPeer(tp.PeerConfig{
		CountTime:  true,
		ListenPort: 9090,
	})
	srv.RoutePull(new(test))
	srv.ListenAndServe()
}

type test struct {
	tp.PullCtx
}

func (t *test) Wait3s(arg *string) (string, *tp.Rerror) {
	time.Sleep(3 * time.Second)
	return *arg + " -> OK", nil
}
