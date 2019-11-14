package md5_test

import (
	"testing"
	"time"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/xfer"
	"github.com/henrylee2cn/erpc/v6/xfer/md5"
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
	srv := erpc.NewPeer(erpc.PeerConfig{ListenPort: 9090})
	srv.RouteCall(new(Home))
	go srv.ListenAndServe()
	time.Sleep(1e9)

	// Client
	cli := erpc.NewPeer(erpc.PeerConfig{})
	sess, stat := cli.Dial(":9090")
	if !stat.OK() {
		t.Fatal(stat)
	}
	var result interface{}
	stat = sess.Call("/home/test",
		map[string]interface{}{
			"bytes": "test bytes",
		},
		&result,
		// Use custom filter
		erpc.WithXferPipe('m'),
	).Status()
	if !stat.OK() {
		t.Error(stat)
	}
	t.Logf("result:%v", result)
}

type Home struct {
	erpc.CallCtx
}

func (h *Home) Test(arg *map[string]interface{}) (map[string]interface{}, *erpc.Status) {
	return map[string]interface{}{
		"result": "your request is:" + (*arg)["bytes"].(string),
	}, nil
}
