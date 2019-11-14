package main

import (
	"fmt"
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
	srv.RouteCallFunc((*ctrl).math_add1)
	srv.RouteCallFunc(math_add2)
	srv.ListenAndServe()
}

type ctrl struct {
	erpc.CallCtx
}

func (c *ctrl) math_add1(arg *[]int) (int, *erpc.Status) {
	return math_add2(c, arg)
}

func math_add2(ctx erpc.CallCtx, arg *[]int) (int, *erpc.Status) {
	if string(ctx.PeekMeta("push_status")) == "yes" {
		ctx.Session().Push(
			"/server/status1",
			fmt.Sprintf("%d numbers are being added...", len(*arg)),
		)
		ctx.Session().Push(
			"/server/status2",
			fmt.Sprintf("%d numbers are being added...", len(*arg)),
		)
		time.Sleep(time.Millisecond * 10)
	}
	var r int
	for _, a := range *arg {
		r += a
	}
	return r, nil
}
