package main

import (
	"encoding/json"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	go tp.GraceSignal()
	// tp.SetReadLimit(10)
	tp.SetShutdown(time.Second*20, nil, nil)
	var peer = tp.NewPeer(tp.PeerConfig{
		SlowCometDuration: time.Millisecond * 500,
		PrintDetail:       true,
		CountTime:         true,
		ListenAddress:     "0.0.0.0:9090",
	})
	group := peer.SubRoute("group")
	group.RoutePull(new(Home))
	peer.SetUnknownPull(UnknownPullHandle)
	peer.ListenAndServe()
}

// Home controller
type Home struct {
	tp.PullCtx
}

// Test handler
func (h *Home) Test(arg *map[string]interface{}) (map[string]interface{}, *tp.Rerror) {
	h.Session().Push("/push/test?tag=from home-test", map[string]interface{}{
		"your_id": h.Query().Get("peer_id"),
	})
	meta := h.CopyMeta()
	meta.VisitAll(func(k, v []byte) {
		tp.Infof("meta: key: %s, value: %s", k, v)
	})
	time.Sleep(5e9)
	return map[string]interface{}{
		"your_arg":    *arg,
		"server_time": time.Now(),
		"meta":        meta.String(),
	}, nil
}

// UnknownPullHandle handles unknown pull packet
func UnknownPullHandle(ctx tp.UnknownPullCtx) (interface{}, *tp.Rerror) {
	time.Sleep(1)
	var v = struct {
		RawMessage json.RawMessage
		Bytes      []byte
	}{}
	codecId, err := ctx.Bind(&v)
	if err != nil {
		return nil, tp.NewRerror(1001, "bind error", err.Error())
	}
	tp.Debugf("UnknownPullHandle: codec: %d, RawMessage: %s, bytes: %s",
		codecId, v.RawMessage, v.Bytes,
	)
	ctx.Session().Push("/push/test?tag=from home-test", map[string]interface{}{
		"your_id": ctx.Query().Get("peer_id"),
	})
	return map[string]interface{}{
		"your_arg":    v,
		"server_time": time.Now(),
		"meta":        ctx.CopyMeta().String(),
	}, nil
}
