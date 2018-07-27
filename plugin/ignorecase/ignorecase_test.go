package ignorecase_test

import (
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/plugin/ignorecase"
)

type Home struct {
	tp.CallCtx
}

func (h *Home) Test(arg *map[string]interface{}) (map[string]interface{}, *tp.Rerror) {
	h.Session().Push("/push/tesT", map[string]interface{}{
		"your_id": h.Query().Get("peer_id"),
	})
	meta := h.CopyMeta()
	time.Sleep(5e9)

	return map[string]interface{}{
		"arg":  *arg,
		"meta": meta.String(),
	}, nil
}

func TestIngoreCase(t *testing.T) {
	// Server
	srv := tp.NewPeer(tp.PeerConfig{ListenPort: 9090}, ignorecase.NewIgnoreCase())
	srv.RouteCall(new(Home))
	go srv.ListenAndServe()
	time.Sleep(1e9)

	// Client
	cli := tp.NewPeer(tp.PeerConfig{}, ignorecase.NewIgnoreCase())
	cli.RoutePush(new(Push))
	sess, err := cli.Dial(":9090")
	if err != nil {
		if err != nil {
			t.Error(err)
		}
	}
	var result interface{}
	rerr := sess.Call("/home/tesT?peer_id=110",
		map[string]interface{}{
			"bytes": []byte("test bytes"),
		},
		&result,
		tp.WithAddMeta("add", "1"),
	).Rerror()
	if rerr != nil {
		t.Error(rerr)
	}
	t.Logf("result:%v", result)
}

type Push struct {
	tp.PushCtx
}

func (p *Push) Test(arg *map[string]interface{}) *tp.Rerror {
	tp.Infof("receive push(%s):\narg: %#v\n", p.Ip(), arg)
	return nil
}
