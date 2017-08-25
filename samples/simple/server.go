package main

import (
	"time"

	"github.com/henrylee2cn/teleport"
)

func main() {
	go teleport.GraceSignal()
	teleport.SetShutdown(time.Second*20, nil, nil)
	var cfg = &teleport.PeerConfig{
		DefaultReadTimeout:  time.Minute * 3,
		DefaultWriteTimeout: time.Minute * 3,
		TlsCertFile:         "",
		TlsKeyFile:          "",
		SlowCometDuration:   time.Millisecond * 500,
		DefaultCodec:        "json",
		DefaultGzipLevel:    5,
		PrintBody:           true,
		ListenAddrs: []string{
			"0.0.0.0:9090",
			"0.0.0.0:9091",
		},
	}
	var peer = teleport.NewPeer(cfg)
	{
		group := peer.PullRouter.Group("group")
		group.Reg(new(Home))
	}
	peer.PullRouter.SetUnknown(UnknownPullHandle)
	peer.Listen()
}

// Home controller
type Home struct {
	teleport.PullCtx
}

// Test handler
func (h *Home) Test(args *map[string]interface{}) (map[string]interface{}, teleport.Xerror) {
	h.Session().Push("/push/test?tag=from home-test", map[string]interface{}{
		"your_id": h.Query().Get("peer_id"),
		"a":       1,
	})
	// time.Sleep(10e9)
	return map[string]interface{}{
		"your_args":   *args,
		"server_time": time.Now(),
	}, nil
}

func UnknownPullHandle(ctx teleport.UnknownPullCtx, body *[]byte) (interface{}, teleport.Xerror) {
	var v interface{}
	codecName, err := ctx.Unmarshal(*body, &v, true)
	if err != nil {
		return nil, teleport.NewXerror(0, err.Error())
	}
	teleport.Debugf("unmarshal body: codec: %s, content: %#v", codecName, v)
	return []string{"a", "aa", "aaa"}, nil
}
