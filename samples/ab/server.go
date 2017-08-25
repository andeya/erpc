package main

import (
	// "net/http"
	// _ "net/http/pprof"
	"time"

	"github.com/henrylee2cn/teleport"
)

func main() {
	// go func() {
	// 	http.ListenAndServe("0.0.0.0:9091", nil)
	// }()
	teleport.SetRawlogLevel("error")
	go teleport.GraceSignal()
	teleport.SetShutdown(time.Second*20, nil, nil)
	var cfg = &teleport.PeerConfig{
		DefaultReadTimeout:  time.Minute * 1,
		DefaultWriteTimeout: time.Minute * 1,
		TlsCertFile:         "",
		TlsKeyFile:          "",
		SlowCometDuration:   time.Millisecond * 500,
		DefaultCodec:        "json",
		DefaultGzipLevel:    0,
		PrintBody:           false,
		ListenAddrs: []string{
			"0.0.0.0:9090",
		},
	}
	var peer = teleport.NewPeer(cfg)
	{
		group := peer.PullRouter.Group("group")
		group.Reg(new(Home))
	}
	peer.Listen()
}

// Home controller
type Home struct {
	teleport.PullCtx
}

// Test handler
func (h *Home) Test(args *[2]int) (int, teleport.Xerror) {
	a := (*args)[0]
	b := (*args)[1]
	return a + b, nil
}
