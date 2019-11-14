package main

import (
	"time"

	"github.com/henrylee2cn/erpc/v6"
)

//go:generate go build $GOFILE

func main() {
	defer erpc.FlushLogger()
	srv := erpc.NewPeer(erpc.PeerConfig{
		CountTime:  true,
		ListenPort: 9090,
	})
	srv.RouteCall(new(test))
	srv.ListenAndServe()
}

type test struct {
	erpc.CallCtx
}

func (t *test) Wait3s(arg *string) (string, *erpc.Status) {
	time.Sleep(3 * time.Second)
	return *arg + " -> OK", nil
}
