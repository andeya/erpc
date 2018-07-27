package clientsession_test

import (
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/mixer/clientsession"
)

type Arg struct {
	A int
	B int `param:"<range:1:>"`
}

type P struct{ tp.CallCtx }

func (p *P) Divide(arg *Arg) (int, *tp.Rerror) {
	return arg.A / arg.B, nil
}

func TestCliSession(t *testing.T) {
	srv := tp.NewPeer(tp.PeerConfig{
		ListenPort: 9090,
	})
	srv.RouteCall(new(P))
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := clientsession.New(
		tp.NewPeer(tp.PeerConfig{}),
		":9090",
		100,
		time.Second*5,
	)
	go func() {
		for {
			t.Logf("%+v", cli.Stats())
			time.Sleep(time.Millisecond * 500)
		}
	}()
	go func() {
		var result int
		for i := 0; ; i++ {
			rerr := cli.Call("/p/divide", &Arg{
				A: i,
				B: 2,
			}, &result).Rerror()
			if rerr != nil {
				t.Log(rerr)
			} else {
				t.Logf("%d/2=%v", i, result)
			}
			time.Sleep(time.Millisecond * 500)
		}
	}()
	time.Sleep(time.Second * 6)
	cli.Close()
	time.Sleep(time.Second * 3)
}
