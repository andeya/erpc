package main

import (
	"context"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	svr := tp.NewPeer(tp.PeerConfig{
		CountTime:         true,
		ListenAddress:     ":9090",
		DefaultSessionAge: time.Second * 7,
		DefaultContextAge: time.Second * 2,
	})
	svr.RoutePull(new(test))
	svr.Listen()
}

type test struct {
	tp.PullCtx
}

func (t *test) Ok(args *string) (string, *tp.Rerror) {
	return *args + " -> OK", nil
}

func (t *test) Timeout(args *string) (string, *tp.Rerror) {
	tCtx, _ := context.WithTimeout(t.Context(), time.Second)
	select {
	case <-tCtx.Done():
		return *args + " -> Not Timeout", nil
		return "", tp.NewRerror(
			tp.CodeHandleTimeout,
			tp.CodeText(tp.CodeHandleTimeout),
			tCtx.Err().Error(),
		)
	default:
	}
	return *args + " -> Not Timeout", nil
}
