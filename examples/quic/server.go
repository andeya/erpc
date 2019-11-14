package main

import (
	"fmt"
	"time"

	"github.com/henrylee2cn/erpc/v6"
)

//go:generate go build $GOFILE

func main() {
	defer erpc.FlushLogger()
	// graceful
	go erpc.GraceSignal()

	// server peer
	srv := erpc.NewPeer(erpc.PeerConfig{
		Network:     "quic",
		CountTime:   true,
		ListenPort:  9090,
		PrintDetail: true,
	})
	e := srv.SetTLSConfigFromFile("cert.pem", "key.pem")
	if e != nil {
		erpc.Fatalf("%v", e)
	}

	// router
	srv.RouteCall(new(Math))

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
	select {}
}

// Math handler
type Math struct {
	erpc.CallCtx
}

// Add handles addition request
func (m *Math) Add(arg *[]int) (int, *erpc.Status) {
	// test query parameter
	erpc.Infof("author: %s", m.PeekMeta("author"))
	// add
	var r int
	for _, a := range *arg {
		r += a
	}
	// response
	return r, nil
}
