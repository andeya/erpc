package config

import (
	"testing"

	"github.com/andeya/cfgo"
	"github.com/andeya/erpc/v7"
	"github.com/andeya/goutil"
)

//go:generate go test -v -c -o "${GOPACKAGE}_server" $GOFILE

func TestServer(t *testing.T) {
	if goutil.IsGoTest() {
		t.Log("skip test in go test")
		return
	}

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
