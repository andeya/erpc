package main

import (
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	go tp.GraceSignal()
	tp.SetShutdown(time.Second*20, nil, nil)
	var peer = tp.NewPeer(tp.PeerConfig{
		SlowCometDuration: time.Millisecond * 500,
		PrintDetail:       true,
		ListenPort:        9090,
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
func (h *Home) Test(arg *[]byte) ([]byte, *tp.Rerror) {
	h.Session().Push("/push/test?tag=from home-test", []byte("test push text"))
	tp.Debugf("HomePullHandle: codec: %d, arg: %s", h.GetBodyCodec(), *arg)
	return []byte("test pull result text"), nil
}

// UnknownPullHandle handles unknown pull packet
func UnknownPullHandle(ctx tp.UnknownPullCtx) (interface{}, *tp.Rerror) {
	ctx.Session().Push("/push/test?tag=from unknown", []byte("test unknown push text"))
	var arg []byte
	codecId, err := ctx.Bind(&arg)
	if err != nil {
		return nil, tp.NewRerror(1001, "bind error", err.Error())
	}
	tp.Debugf("UnknownPullHandle: codec: %d, arg: %s", codecId, arg)
	return []byte("test unknown pull result text"), nil
}
