package main

import (
	"context"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	srv := tp.NewPeer(tp.PeerConfig{
		PrintDetail:       true,
		CountTime:         true,
		ListenPort:        9090,
		DefaultSessionAge: time.Second * 7,
		DefaultContextAge: time.Second * 2,
	})
	srv.RoutePull(new(test))
	srv.ListenAndServe()
}

type test struct {
	tp.PullCtx
}

func (t *test) Ok(arg *string) (string, *tp.Rerror) {
	return *arg + " -> OK", nil
}

func (t *test) Timeout(arg *string) (string, *tp.Rerror) {
	tCtx, _ := context.WithTimeout(t.Context(), time.Second)
	time.Sleep(time.Second)
	select {
	case <-tCtx.Done():
		return "", tp.NewRerror(
			tp.CodeHandleTimeout,
			tp.CodeText(tp.CodeHandleTimeout),
			tCtx.Err().Error(),
		)
	default:
	}
	return *arg + " -> Not Timeout", nil
}
