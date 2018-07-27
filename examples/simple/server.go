package main

import (
	"fmt"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	srv := tp.NewPeer(tp.PeerConfig{
		CountTime:   true,
		ListenPort:  9090,
		PrintDetail: true,
	})
	srv.RouteCall(new(math))
	srv.ListenAndServe()
}

type math struct {
	tp.CallCtx
}

func (m *math) Add(arg *[]int) (int, *tp.Rerror) {
	if m.Query().Get("push_status") == "yes" {
		m.Session().Push(
			"/push/status",
			fmt.Sprintf("numbers %+v are being added...", arg),
		)
		time.Sleep(time.Millisecond * 10)
	}
	var r int
	for _, a := range *arg {
		r += a
	}
	return r, nil
}
