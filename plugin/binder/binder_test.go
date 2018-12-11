package binder_test

import (
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/plugin/binder"
)

type (
	Arg struct {
		A int
		B int    `param:"<range:1:100>"`
		C string `param:"<regexp:^[1-9]\\d*$>"`
		Query
		XyZ       string  `param:"<meta><nonzero><rerr: 100002: Parameter cannot be empty>"`
		SwapValue float32 `param:"<swap><nonzero>"`
	}
	Query struct {
		X string `param:"<meta:_x>"`
	}
)

type P struct{ tp.CallCtx }

func (p *P) Divide(arg *Arg) (int, *tp.Rerror) {
	tp.Infof("meta arg _x: %s, xy_z: %s, swap_value: %v", arg.Query.X, arg.XyZ, arg.SwapValue)
	return arg.A / arg.B, nil
}

type SwapPlugin struct{}

func (s *SwapPlugin) Name() string {
	return "swap_plugin"
}
func (s *SwapPlugin) PostReadCallBody(ctx tp.ReadCtx) *tp.Rerror {
	ctx.Swap().Store("swap_value", 123)
	return nil
}

func TestBinder(t *testing.T) {
	bplugin := binder.NewStructArgsBinder(nil)
	srv := tp.NewPeer(
		tp.PeerConfig{ListenPort: 9090},
	)
	srv.PluginContainer().AppendRight(bplugin)
	srv.RouteCall(new(P), new(SwapPlugin))
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := tp.NewPeer(tp.PeerConfig{})
	sess, err := cli.Dial(":9090")
	if err != nil {
		t.Fatal(err)
	}
	var result int
	rerr := sess.Call("/p/divide", &Arg{
		A: 10,
		B: 2,
		C: "3",
	},
		&result,
		tp.WithSetMeta("_x", "testmeta_x"),
		tp.WithSetMeta("xy_z", "testmeta_xy_z"),
	).Rerror()
	if rerr != nil {
		t.Fatal(rerr)
	}
	t.Logf("10/2=%d", result)
	rerr = sess.Call("/p/divide", &Arg{
		A: 10,
		B: 5,
		C: "3",
	}, &result).Rerror()
	if rerr == nil {
		t.Fatal(rerr)
	}
	t.Logf("10/5 error:%v", rerr)
	rerr = sess.Call("/p/divide", &Arg{
		A: 10,
		B: 0,
		C: "3",
	}, &result).Rerror()
	if rerr == nil {
		t.Fatal(rerr)
	}
	t.Logf("10/0 error:%v", rerr)
}
