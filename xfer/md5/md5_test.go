package md5_test

import (
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/xfer"
	"github.com/henrylee2cn/teleport/xfer/md5"
)

func TestSeparate(t *testing.T) {
	// Register filter(custom)
	md5.Reg('m', "md5")
	md5Check, _ := xfer.Get('m')
	input := []byte("md5")
	b, err := md5Check.OnPack(input)
	if err != nil {
		t.Fatalf("Onpack: %v", err)
	}

	_, err = md5Check.OnUnpack(b)
	if err != nil {
		t.Fatalf("Md5 check failed: %v", err)
	}

	// Tamper with data
	b = append(b, "viruses"...)
	_, err = md5Check.OnUnpack(b)
	if err == nil {
		t.Fatal("Md5 check failed:")
	}
}

func TestCombined(t *testing.T) {
	// Register filter(custom)
	md5.Reg('m', "md5")
	// Server
	srv := tp.NewPeer(tp.PeerConfig{ListenPort: 9090})
	srv.RouteCall(new(Home))
	go srv.ListenAndServe()
	time.Sleep(1e9)

	// Client
	cli := tp.NewPeer(tp.PeerConfig{})
	sess, err := cli.Dial(":9090")
	if err != nil {
		if err != nil {
			t.Error(err)
		}
	}
	var result interface{}
	rerr := sess.Call("/home/test?peer_id=110",
		map[string]interface{}{
			"bytes": []byte("test bytes"),
		},
		&result,
		// Use custom filter
		tp.WithXferPipe('m'),
	).Rerror()
	if rerr != nil {
		t.Error(rerr)
	}
	t.Logf("=========result:%v", result)
}

type Home struct {
	tp.CallCtx
}

func (h *Home) Test(arg *map[string]interface{}) (map[string]interface{}, *tp.Rerror) {
	return map[string]interface{}{
		"your_id": h.Query().Get("peer_id"),
	}, nil
}
