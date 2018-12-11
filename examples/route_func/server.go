package main

import (
	"fmt"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

//go:generate go build $GOFILE

func main() {
	defer tp.FlushLogger()
	srv := tp.NewPeer(tp.PeerConfig{
		CountTime:  true,
		ListenPort: 9090,
	})
	srv.RouteCallFunc((*ctrl).math_add1)
	srv.RouteCallFunc(math_add2)
	srv.ListenAndServe()
}

type ctrl struct {
	tp.CallCtx
}

func (c *ctrl) math_add1(arg *[]int) (int, *tp.Rerror) {
	return math_add2(c, arg)
}

func math_add2(ctx tp.CallCtx, arg *[]int) (int, *tp.Rerror) {
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
