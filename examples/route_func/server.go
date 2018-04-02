package main

import (
	"fmt"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	srv := tp.NewPeer(tp.PeerConfig{
		CountTime:     true,
		ListenAddress: ":9090",
	})
	srv.RoutePullFunc((*ctrl).math_add1)
	srv.RoutePullFunc(math_add2)
	srv.ListenAndServe()
}

type ctrl struct {
	tp.PullCtx
}

func (c *ctrl) math_add1(args *[]int) (int, *tp.Rerror) {
	return math_add2(c, args)
}

func math_add2(ctx tp.PullCtx, args *[]int) (int, *tp.Rerror) {
	if ctx.Query().Get("push_status") == "yes" {
		ctx.Session().Push(
			"/server/status1",
			fmt.Sprintf("%d numbers are being added...", len(*args)),
		)
		ctx.Session().Push(
			"/server/status2",
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
