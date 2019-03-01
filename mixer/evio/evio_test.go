package evio_test

import (
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/mixer/evio"
)

func Test(t *testing.T) {
	// server
	srv := evio.NewServer(1, tp.PeerConfig{ListenPort: 9090})
	srv.RouteCall(new(Home))
	go srv.ListenAndServe()
	time.Sleep(1e9)

	// client
	cli := evio.NewClient(tp.PeerConfig{})
	cli.RoutePush(new(Push))
	sess, err := cli.Dial(":9090")
	if err != nil {
		t.Error(err)
	}
	var result interface{}
	rerr := sess.Call("/home/test",
		map[string]string{
			"author": "henrylee2cn",
		},
		&result,
		tp.WithAddMeta("peer_id", "110"),
	).Rerror()
	if rerr != nil {
		t.Error(rerr)
	}
	t.Logf("result:%v", result)
	time.Sleep(2e9)
}

type Home struct {
	tp.CallCtx
}

func (h *Home) Test(arg *map[string]string) (map[string]interface{}, *tp.Rerror) {
	h.Session().Push("/push/test", map[string]string{
		"your_id": string(h.PeekMeta("peer_id")),
	})
	return map[string]interface{}{
		"arg": *arg,
	}, nil
}

type Push struct {
	tp.PushCtx
}

func (p *Push) Test(arg *map[string]string) *tp.Rerror {
	tp.Infof("receive push(%s):\narg: %#v\n", p.IP(), arg)
	return nil
}
