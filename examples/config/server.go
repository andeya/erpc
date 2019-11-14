package main

import (
	"github.com/henrylee2cn/cfgo"
	"github.com/henrylee2cn/erpc/v6"
)

//go:generate go build $GOFILE

func main() {
	defer erpc.FlushLogger()
	go erpc.GraceSignal()
	cfg := erpc.PeerConfig{
		CountTime:  true,
		ListenPort: 9090,
	}

	// auto create and sync config/config.yaml
	cfgo.MustGet("config/config.yaml", true).MustReg("cfg_srv", &cfg)

	srv := erpc.NewPeer(cfg)
	srv.RouteCall(new(math))
	srv.ListenAndServe()
}

type math struct {
	erpc.CallCtx
}

func (m *math) Add(arg *[]int) (int, *erpc.Status) {
	var r int
	for _, a := range *arg {
		r += a
	}
	return r, nil
}
