package main

import (
	tp "github.com/henrylee2cn/teleport"
)

func main() {
	svr := tp.NewPeer(tp.PeerConfig{
		CountTime:     true,
		ListenAddress: ":9090",
	})
	svr.RoutePull(new(math))
	svr.RoutePush(new(chat))
	svr.Listen()
}

type math struct {
	tp.PullCtx
}

func (m *math) Add(args *[]int) (int, *tp.Rerror) {
	var r int
	for _, a := range *args {
		r += a
	}
	return r, nil
}

type chat struct {
	tp.PushCtx
}

func (c *chat) Say(args *string) *tp.Rerror {
	tp.Printf("%s say: %q", c.RealId(), *args)
	return nil
}
