package secure_test

import (
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/plugin/secure"
)

type Arg struct {
	A int
	B int
}

type Result struct {
	C int
}

type math struct{ tp.CallCtx }

func (m *math) Add(arg *Arg) (*Result, *tp.Rerror) {
	// enforces the body of the encrypted reply packet.
	// secure.EnforceSecure(m.Output())

	return &Result{C: arg.A + arg.B}, nil
}

func newSession(t *testing.T) tp.Session {
	p := secure.NewSecurePlugin(100001, "cipherkey1234567")
	srv := tp.NewPeer(tp.PeerConfig{
		ListenPort:  9090,
		PrintDetail: true,
	})
	srv.RouteCall(new(math), p)
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := tp.NewPeer(tp.PeerConfig{
		PrintDetail: true,
	}, p)
	sess, err := cli.Dial(":9090")
	if err != nil {
		t.Fatal(err)
	}
	return sess
}

func TestSecurePlugin(t *testing.T) {
	sess := newSession(t)
	// test secure
	var result Result
	rerr := sess.Call(
		"/math/add",
		&Arg{A: 10, B: 2},
		&result,
		secure.WithSecureMeta(),
		// secure.WithAcceptSecureMeta(false),
	).Rerror()
	if rerr != nil {
		t.Fatal(rerr)
	}
	if result.C != 12 {
		t.Fatalf("expect 12, but get %d", result.C)
	}
	t.Logf("test secure10+2=%d", result.C)
}

func TestAcceptSecurePlugin(t *testing.T) {
	sess := newSession(t)
	// test accept secure
	var result Result
	rerr := sess.Call(
		"/math/add",
		&Arg{A: 20, B: 4},
		&result,
		secure.WithAcceptSecureMeta(true),
	).Rerror()
	if rerr != nil {
		t.Fatal(rerr)
	}
	if result.C != 24 {
		t.Fatalf("expect 24, but get %d", result.C)
	}
	t.Logf("test accept secure: 20+4=%d", result.C)
}
