package proxy_and_seq

import (
	"testing"
	"time"

	"github.com/andeya/erpc/v7"
	"github.com/andeya/erpc/v7/plugin/proxy"
	"github.com/andeya/goutil"
)

//go:generate go test -v -c -o "${GOPACKAGE}" $GOFILE

func TestProxy(t *testing.T) {
	if goutil.IsGoTest() {
		t.Log("skip test in go test")
		return
	}
	defer erpc.FlushLogger()
	srv := erpc.NewPeer(
		erpc.PeerConfig{
			ListenPort: 8080,
		},
		newProxyPlugin(),
	)
	srv.ListenAndServe()
}

func newProxyPlugin() erpc.Plugin {
	cli := erpc.NewPeer(erpc.PeerConfig{RedialTimes: 3})
	var sess erpc.Session
	var stat *erpc.Status
DIAL:
	sess, stat = cli.Dial(":9090")
	if !stat.OK() {
		erpc.Warnf("%v", stat)
		time.Sleep(time.Second * 3)
		goto DIAL
	}
	return proxy.NewPlugin(func(*proxy.Label) proxy.Forwarder {
		return sess
	})
}
