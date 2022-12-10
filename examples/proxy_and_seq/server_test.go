package proxy_and_seq

import (
	"testing"

	"github.com/andeya/erpc/v7"
	"github.com/andeya/goutil"
)

//go:generate go test -v -c -o "${GOPACKAGE}_server" $GOFILE

func TestServer(t *testing.T) {
	if goutil.IsGoTest() {
		t.Log("skip test in go test")
		return
	}
	defer erpc.FlushLogger()
	srv := erpc.NewPeer(erpc.PeerConfig{
		CountTime:  true,
		ListenPort: 9090,
	})
	srv.RouteCall(new(math))
	srv.RoutePush(new(chat))
	srv.ListenAndServe()
}

type math struct {
	erpc.CallCtx
}

func (m *math) Add(arg *[]int) (int, *erpc.Status) {
	var r int
	for _, a := range *arg {
		r += a
	}
	return r, nil
}

type chat struct {
	erpc.PushCtx
}

func (c *chat) Say(arg *string) *erpc.Status {
	erpc.Printf("%s say: %q", c.PeekMeta("X-ID"), *arg)
	return nil
}
