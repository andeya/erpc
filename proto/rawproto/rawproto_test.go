package rawproto_test

import (
	"testing"
	"time"

	"github.com/andeya/erpc/v7"
	"github.com/andeya/erpc/v7/xfer/gzip"
	"github.com/andeya/goutil"
)

//go:generate go test -v -c -o "${GOPACKAGE}" $GOFILE

func TestRawProto(t *testing.T) {
	if goutil.IsGoTest() {
		t.Log("skip test in go test")
		return
	}

	gzip.Reg('g', "gizp-5", 5)

	// server
	srv := erpc.NewPeer(erpc.PeerConfig{ListenPort: 9090})
	srv.RouteCall(new(Home))
	go srv.ListenAndServe()
	time.Sleep(1e9)

	// client
	cli := erpc.NewPeer(erpc.PeerConfig{})
	cli.RoutePush(new(Push))
	sess, stat := cli.Dial(":9090")
	if !stat.OK() {
		t.Fatal(stat)
	}
	var result interface{}
	stat = sess.Call("/home/test",
		map[string]string{
			"author": "andeya",
		},
		&result,
		erpc.WithAddMeta("peer_id", "110"),
		erpc.WithXferPipe('g'),
	).Status()
	if !stat.OK() {
		t.Error(stat)
	}
	t.Logf("result:%v", result)
	time.Sleep(3e9)
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
