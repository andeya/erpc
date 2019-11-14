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
	group := srv.SubRoute("/srv")
	group.RouteCall(new(math_v2))
	srv.ListenAndServe()
}

type math_v2 struct {
	erpc.CallCtx
}

func (m *math_v2) Add__2(arg *[]int) (int, *erpc.Status) {
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
