package evio_test

import (
	"testing"
	"time"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/mixer/evio"
)

func Test(t *testing.T) {
	// server
	srv := evio.NewServer(1, erpc.PeerConfig{ListenPort: 9090})
	// use TLS
	srv.SetTLSConfig(erpc.GenerateTLSConfigForServer())
	srv.RouteCall(new(Home))
	go srv.ListenAndServe()
	time.Sleep(1e9)

	// client
	cli := evio.NewClient(erpc.PeerConfig{})
	// use TLS
	cli.SetTLSConfig(erpc.GenerateTLSConfigForClient())
	cli.RoutePush(new(Push))
	sess, stat := cli.Dial(":9090")
	if !stat.OK() {
		t.Fatal(stat)
	}
	var result interface{}
	stat = sess.Call("/home/test",
		map[string]string{
			"author": "henrylee2cn",
		},
		&result,
		erpc.WithAddMeta("peer_id", "110"),
	).Status()
	if !stat.OK() {
		t.Fatal(stat)
	}
	t.Logf("result:%v", result)
	time.Sleep(2e9)
}

type Home struct {
	erpc.CallCtx
}

func (h *Home) Test(arg *map[string]string) (map[string]interface{}, *erpc.Status) {
	h.Session().Push("/push/test", map[string]string{
		"your_id": string(h.PeekMeta("peer_id")),
	})
	return map[string]interface{}{
		"arg": *arg,
	}, nil
}

type Push struct {
	erpc.PushCtx
}

func (p *Push) Test(arg *map[string]string) *erpc.Status {
	erpc.Infof("receive push(%s):\narg: %#v\n", p.IP(), arg)
	return nil
}
