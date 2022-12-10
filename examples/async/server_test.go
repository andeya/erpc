package async

import (
	"testing"
	"time"

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
	srv.RouteCall(new(test))
	srv.ListenAndServe()
}

type test struct {
	erpc.CallCtx
}

func (t *test) Wait3s(arg *string) (string, *erpc.Status) {
	time.Sleep(3 * time.Second)
	return *arg + " -> OK", nil
}
