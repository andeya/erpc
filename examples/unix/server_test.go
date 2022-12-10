package unix

import (
	"os"
	"testing"

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
	srv := erpc.NewPeer(erpc.PeerConfig{
		CountTime:  true,
		ListenPort: 9090,
		Network:    "unix",
	}, &erpc.PluginImpl{
		PluginName: "clean-listen-unix-file",
		OnPreNewPeer: func(config *erpc.PeerConfig, _ *erpc.PluginContainer) error {
			socketFile := config.ListenAddr().String()
			if _, err := os.Stat(socketFile); err == nil {
				if err := os.RemoveAll(socketFile); err != nil {
					return err
				}
			}
			return nil
		},
	})
	srv.RouteCall(&Echo{})
	srv.ListenAndServe()
}

type Echo struct {
	erpc.CallCtx
}

func (echo *Echo) Parrot(arg *string) (string, *erpc.Status) {
	return *arg, nil
}
