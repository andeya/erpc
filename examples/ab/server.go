package main

import (
	// "net/http"
	// _ "net/http/pprof"
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/socket/example/pb"
)

func main() {
	// go func() {
	// 	http.ListenAndServe("0.0.0.0:9091", nil)
	// }()
	tp.SetSocketNoDelay(false)
	tp.SetLoggerLevel("WARNING")
	go tp.GraceSignal()
	tp.SetShutdown(time.Second*20, nil, nil)
	var peer = tp.NewPeer(tp.PeerConfig{
		DefaultBodyCodec: "protobuf",
		ListenPort:       9090,
	})
	{
		group := peer.SubRoute("group")
		group.RouteCall(new(Home))
	}
	peer.ListenAndServe()
}

// Home controller
type Home struct {
	tp.CallCtx
}

// Test handler
func (h *Home) Test(arg *pb.PbTest) (*pb.PbTest, *tp.Rerror) {
	return &pb.PbTest{
		A: arg.A + arg.B,
		B: arg.A - arg.B,
	}, nil
}
