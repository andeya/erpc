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
	group := srv.SubRoute("/srv")
	group.RouteCall(new(math_v2))
	srv.ListenAndServe()
}

type math_v2 struct {
	tp.CallCtx
}

func (m *math_v2) Add__2(arg *[]int) (int, *tp.Rerror) {
	if string(m.PeekMeta("push_status")) == "yes" {
		m.Session().Push(
			"/cli/push/server_status",
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
