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
	group := svr.SubRoute("/srv")
	group.RoutePull(new(math))
	svr.Listen()
}

type math struct {
	tp.PullCtx
}

func (m *math) Add(args *[]int) (int, *tp.Rerror) {
	if m.Query().Get("push_status") == "yes" {
		m.Session().Push(
			"/cli/push/status",
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
