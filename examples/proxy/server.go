package main

import (
	tp "github.com/henrylee2cn/teleport"
)

func main() {
	srv := tp.NewPeer(tp.PeerConfig{
		CountTime:     true,
		ListenAddress: ":9090",
	})
	srv.RoutePull(new(math))
	srv.RoutePush(new(chat))
	srv.ListenAndServe()
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
	tp.Printf("%s say: %q", c.PeekMeta("X-ID"), *args)
	return nil
}
