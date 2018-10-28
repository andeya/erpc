package main

import (
	tp "github.com/henrylee2cn/teleport"
)

//go:generate go build $GOFILE

func main() {
	defer tp.FlushLogger()
	srv := tp.NewPeer(tp.PeerConfig{
		CountTime:  true,
		ListenPort: 9090,
	})
	srv.RouteCall(new(math))
	srv.RoutePush(new(chat))
	srv.ListenAndServe()
}

type math struct {
	tp.CallCtx
}

func (m *math) Add(arg *[]int) (int, *tp.Rerror) {
	var r int
	for _, a := range *arg {
		r += a
	}
	return r, nil
}

type chat struct {
	tp.PushCtx
}

func (c *chat) Say(arg *string) *tp.Rerror {
	tp.Printf("%s say: %q", c.PeekMeta("X-ID"), *arg)
	return nil
}
