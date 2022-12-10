package twoway

import (
	"fmt"
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
	// graceful
	go erpc.GraceSignal()

	// server peer
	srv := erpc.NewPeer(erpc.PeerConfig{
		CountTime:   true,
		ListenPort:  9090,
		PrintDetail: true,
	})
	srv.SetTLSConfig(erpc.GenerateTLSConfigForServer())

	// router
	srv.RouteCall(new(Math))
	group := srv.SubRoute("/srv")
	group.RouteCall(new(math_v2))

	// broadcast per 5s
	go func() {
		for {
			time.Sleep(time.Second * 5)
			srv.RangeSession(func(sess erpc.Session) bool {
				sess.Push(
					"/push/status",
					fmt.Sprintf("this is a broadcast, server time: %v", time.Now()),
				)
				return true
			})
		}
	}()

	// listen and serve
	srv.ListenAndServe()
}

// Math handler
type Math struct {
	erpc.CallCtx
}

// Add handles addition request
func (m *Math) Add(arg *[]int) (int, *erpc.Status) {
	// test meta
	erpc.Infof("author: %s", m.PeekMeta("author"))
	// add
	var r int
	for _, a := range *arg {
		r += a
	}
	// response
	return r, nil
}

type math_v2 struct {
	erpc.CallCtx
}

func (m *math_v2) Add__2(arg *[]int) (int, *erpc.Status) {
	if string(m.PeekMeta("push_status")) == "yes" {
		m.Session().Push(
			"/cli/push/server_status",
			fmt.Sprintf("%d numbers are being added...", len(*arg)),
		)
		time.Sleep(time.Millisecond * 10)
	}
	var r int
	for _, a := range *arg {
		r += a
	}
	return r, nil
}
