package thriftproto_test

import (
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/proto/thriftproto"
	"github.com/henrylee2cn/teleport/xfer/gzip"
)

var withoutHeader bool

type Home struct {
	tp.CallCtx
}

func (h *Home) Test(arg *thriftproto.Test) (*thriftproto.Test, *tp.Rerror) {
	if withoutHeader {
		if h.CopyMeta().Len() != 0 {
			panic("except meta is empty")
		}
	} else {
		if string(h.PeekMeta("peer_id")) != "110" {
			panic("except meta: peer_id=110")
		}
	}
	return &thriftproto.Test{
		Author: arg.Author + "->OK",
	}, nil
}

func TestBinaryProto(t *testing.T) {
	withoutHeader = false
	gzip.Reg('g', "gizp-5", 5)

	// server
	srv := tp.NewPeer(tp.PeerConfig{ListenPort: 9090, DefaultBodyCodec: "thrift"})
	srv.RouteCall(new(Home))
	go srv.ListenAndServe(thriftproto.NewBinaryProtoFunc())
	time.Sleep(1e9)

	// client
	cli := tp.NewPeer(tp.PeerConfig{DefaultBodyCodec: "thrift"})
	sess, err := cli.Dial(":9090", thriftproto.NewBinaryProtoFunc())
	if err != nil {
		t.Error(err)
	}
	var result thriftproto.Test
	rerr := sess.Call("Home.Test",
		&thriftproto.Test{Author: "henrylee2cn"},
		&result,
		tp.WithAddMeta("peer_id", "110"),
		tp.WithXferPipe('g'),
	).Rerror()
	if rerr != nil {
		t.Error(rerr)
	}
	if result.Author != "henrylee2cn->OK" {
		t.FailNow()
	}
	t.Logf("result:%v", result)
}

func TestStructProto(t *testing.T) {
	withoutHeader = false
	// server
	srv := tp.NewPeer(tp.PeerConfig{ListenPort: 9090})
	srv.RouteCall(new(Home))
	go srv.ListenAndServe(thriftproto.NewStructProtoFunc(true))
	time.Sleep(1e9)

	// client
	cli := tp.NewPeer(tp.PeerConfig{})
	sess, err := cli.Dial(":9090", thriftproto.NewStructProtoFunc(true))
	if err != nil {
		t.Error(err)
	}
	var result thriftproto.Test
	rerr := sess.Call("Home.Test",
		&thriftproto.Test{Author: "henrylee2cn"},
		&result,
		tp.WithAddMeta("peer_id", "110"),
	).Rerror()
	if rerr != nil {
		t.Error(rerr)
	}
	if result.Author != "henrylee2cn->OK" {
		t.FailNow()
	}
	t.Logf("result:%v", result)
}

func TestStructProtoWithoutHeaders(t *testing.T) {
	withoutHeader = true
	// server
	srv := tp.NewPeer(tp.PeerConfig{ListenPort: 9090})
	srv.RouteCall(new(Home))
	go srv.ListenAndServe(thriftproto.NewStructProtoFunc(false))
	time.Sleep(1e9)

	// client
	cli := tp.NewPeer(tp.PeerConfig{})
	sess, err := cli.Dial(":9090", thriftproto.NewStructProtoFunc(false))
	if err != nil {
		t.Error(err)
	}
	var result thriftproto.Test
	rerr := sess.Call("Home.Test",
		&thriftproto.Test{Author: "henrylee2cn"},
		&result,
		tp.WithAddMeta("peer_id", "110"),
	).Rerror()
	if rerr != nil {
		t.Error(rerr)
	}
	if result.Author != "henrylee2cn->OK" {
		t.FailNow()
	}
	t.Logf("result:%v", result)
}
