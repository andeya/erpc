package main

import (
	"time"

	"github.com/henrylee2cn/erpc/v6"
)

//go:generate go build $GOFILE

func main() {
	defer erpc.FlushLogger()
	go erpc.GraceSignal()
	erpc.SetShutdown(time.Second*20, nil, nil)
	var peer = erpc.NewPeer(erpc.PeerConfig{
		SlowCometDuration: time.Millisecond * 500,
		PrintDetail:       true,
		ListenPort:        9090,
	})
	group := peer.SubRoute("group")
	group.RouteCall(new(Home))
	peer.SetUnknownCall(UnknownCallHandle)
	peer.ListenAndServe()
}

// Home controller
type Home struct {
	erpc.CallCtx
}

// Test handler
func (h *Home) Test(arg *[]byte) ([]byte, *erpc.Status) {
	h.Session().Push("/push/test", []byte("test push text"))
	erpc.Debugf("HomeCallHandle: codec: %d, arg: %s", h.GetBodyCodec(), *arg)
	return []byte("test call result text"), nil
}

// UnknownCallHandle handles unknown call message
func UnknownCallHandle(ctx erpc.UnknownCallCtx) (interface{}, *erpc.Status) {
	ctx.Session().Push("/push/test", []byte("test unknown push text"))
	var arg []byte
	codecID, err := ctx.Bind(&arg)
	if err != nil {
		return nil, erpc.NewStatus(1001, "bind error", err.Error())
	}
	erpc.Debugf("UnknownCallHandle: codec: %d, arg: %s", codecID, arg)
	return []byte("test unknown call result text"), nil
}
