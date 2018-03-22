package main

import (
	"fmt"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	svr := tp.NewPeer(tp.PeerConfig{
		CountTime:     true,
		ListenAddress: ":9090",
	})
	svr.RoutePullFunc((*ctrl).MathAdd1)
	svr.RoutePullFunc(mathAdd2)
	svr.Listen()
}

type ctrl struct {
	tp.PullCtx
}

func (c *ctrl) MathAdd1(args *[]int) (int, *tp.Rerror) {
	return mathAdd2(c, args)
}

func mathAdd2(ctx tp.PullCtx, args *[]int) (int, *tp.Rerror) {
	if ctx.Query().Get("push_status") == "yes" {
		ctx.Session().Push(
			"/status1",
			fmt.Sprintf("%d numbers are being added...", len(*args)),
		)
		ctx.Session().Push(
			"/status2",
			fmt.Sprintf("%d numbers are being added...", len(*args)),
		)
		time.Sleep(time.Millisecond * 10)
	}
	var r int
	for _, a := range *args {
		r += a
	}
	return r, nil
}
