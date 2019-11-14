package main

import (
	"context"
	"time"

	"github.com/henrylee2cn/erpc/v6"
)

//go:generate go build $GOFILE

func main() {
	defer erpc.FlushLogger()
	srv := erpc.NewPeer(erpc.PeerConfig{
		PrintDetail:       true,
		CountTime:         true,
		ListenPort:        9090,
		DefaultSessionAge: time.Second * 7,
		DefaultContextAge: time.Second * 2,
	})
	srv.RouteCall(new(test))
	srv.ListenAndServe()
}

type test struct {
	erpc.CallCtx
}

func (t *test) Ok(arg *string) (string, *erpc.Status) {
	return *arg + " -> OK", nil
}

func (t *test) Timeout(arg *string) (string, *erpc.Status) {
	tCtx, _ := context.WithTimeout(t.Context(), time.Second)
	time.Sleep(time.Second)
	select {
	case <-tCtx.Done():
		return "", erpc.NewStatus(
			erpc.CodeHandleTimeout,
			erpc.CodeText(erpc.CodeHandleTimeout),
			tCtx.Err().Error(),
		)
	default:
		return *arg + " -> Not Timeout", nil
	}
}

func (t *test) Break(*struct{}) (*struct{}, *erpc.Status) {
	time.Sleep(time.Second * 3)
	select {
	case <-t.Session().CloseNotify():
		erpc.Infof("the connection has gone away!")
		return nil, erpc.NewStatus(erpc.CodeConnClosed, "", "")
	default:
		return nil, nil
	}
}
