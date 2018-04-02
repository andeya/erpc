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
	group := srv.SubRoute("/srv")
	group.RoutePull(new(math_v2))
	srv.ListenAndServe()
}

type math_v2 struct {
	tp.PullCtx
}

func (m *math_v2) Add__2(args *[]int) (int, *tp.Rerror) {
	if m.Query().Get("push_status") == "yes" {
		m.Session().Push(
			"/cli/push/server_status",
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
