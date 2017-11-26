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
	tp.SetRawlogLevel("error")
	go tp.GraceSignal()
	tp.SetShutdown(time.Second*20, nil, nil)
	var cfg = &tp.PeerConfig{
		DefaultBodyCodec: "protobuf",
		ListenAddrs: []string{
			"0.0.0.0:9090",
		},
	}
	var peer = tp.NewPeer(cfg)
	{
		group := peer.PullRouter.Group("group")
		group.Reg(new(Home))
	}
	peer.Listen()
}

// Home controller
type Home struct {
	tp.PullCtx
}

// Test handler
func (h *Home) Test(args *pb.PbTest) (*pb.PbTest, *tp.Rerror) {
	return &pb.PbTest{
		A: args.A + args.B,
		B: args.A - args.B,
	}, nil
}
