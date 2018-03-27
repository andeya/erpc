package main

import (
	"github.com/henrylee2cn/cfgo"
	tp "github.com/henrylee2cn/teleport"
)

func main() {
	cfg := tp.PeerConfig{
		CountTime:     true,
		ListenAddress: ":9090",
	}

	// auto create and sync config/config.yaml
	cfgo.MustGet("config/config.yaml", true).MustReg("cfg_srv", &cfg)

	srv := tp.NewPeer(cfg)
	srv.RoutePull(new(math))
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
