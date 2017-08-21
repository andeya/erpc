package main

import (
	"time"

	"github.com/henrylee2cn/teleport"
)

func main() {
	var cfg = &teleport.Config{
		Id:                       "server-peer",
		ReadTimeout:              time.Second * 10,
		WriteTimeout:             time.Second * 10,
		TlsCertFile:              "",
		TlsKeyFile:               "",
		SlowCometDuration:        time.Millisecond * 500,
		DefaultCodec:             "json",
		DefaultGzipLevel:         5,
		MaxGoroutinesAmount:      1024,
		MaxGoroutineIdleDuration: time.Second * 10,
		ListenAddrs: []string{
			"0.0.0.0:9090",
			"0.0.0.0:9091",
		},
	}
	var peer = teleport.NewPeer(cfg)
	{
		group := peer.RequestRouter.Group("group")
		group.Reg(new(Home))
	}
	peer.Listen()
}

// Home controller
type Home struct {
	teleport.RequestCtx
}

// Test handler
func (h *Home) Test(args *string) (string, teleport.Xerror) {
	teleport.Infof("query: %#v", h.Query())
	return "home-test response:" + *args, nil
}
