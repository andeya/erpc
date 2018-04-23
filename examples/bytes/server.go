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
func (h *Home) Test(args *[]byte) ([]byte, *tp.Rerror) {
	h.Session().Push("/push/test?tag=from home-test", []byte("test push text"))
	tp.Debugf("HomePullHandle: codec: %d, args: %s", h.GetBodyCodec(), *args)
	return []byte("test pull reply text"), nil
}

// UnknownPullHandle handles unknown pull packet
func UnknownPullHandle(ctx tp.UnknownPullCtx) (interface{}, *tp.Rerror) {
	ctx.Session().Push("/push/test?tag=from unknown", []byte("test unknown push text"))
	var args []byte
	codecId, err := ctx.Bind(&args)
	if err != nil {
		return nil, tp.NewRerror(1001, "bind error", err.Error())
	}
	tp.Debugf("UnknownPullHandle: codec: %d, args: %s", codecId, args)
	return []byte("test unknown pull reply text"), nil
}
