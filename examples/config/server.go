package main

import (
	"github.com/henrylee2cn/cfgo"
	tp "github.com/henrylee2cn/teleport"
)

//go:generate go build $GOFILE

func main() {
	defer tp.FlushLogger()
	go tp.GraceSignal()
	cfg := tp.PeerConfig{
		CountTime:  true,
		ListenPort: 9090,
	}

	// auto create and sync config/config.yaml
	cfgo.MustGet("config/config.yaml", true).MustReg("cfg_srv", &cfg)

	srv := tp.NewPeer(cfg)
	srv.RouteCall(new(math))
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
