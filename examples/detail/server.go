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
		ListenPort:        9090,
	})
	group := peer.SubRoute("group")
	group.RouteCall(new(Home))
	peer.SetUnknownCall(UnknownCallHandle)
	peer.ListenAndServe()
}

// Home controller
type Home struct {
	tp.CallCtx
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

// UnknownCallHandle handles unknown call packet
func UnknownCallHandle(ctx tp.UnknownCallCtx) (interface{}, *tp.Rerror) {
	time.Sleep(1)
	var v = struct {
		RawMessage json.RawMessage
		Bytes      []byte
	}{}
	codecId, err := ctx.Bind(&v)
	if err != nil {
		return nil, tp.NewRerror(1001, "bind error", err.Error())
	}
	tp.Debugf("UnknownCallHandle: codec: %d, RawMessage: %s, bytes: %s",
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
