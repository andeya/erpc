package main

import (
	"fmt"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

//go:generate go build $GOFILE

func main() {
	defer tp.FlushLogger()
	// graceful
	go tp.GraceSignal()

	// server peer
	srv := tp.NewPeer(tp.PeerConfig{
		Network:     "quic",
		CountTime:   true,
		ListenPort:  9090,
		PrintDetail: true,
	})
	e := srv.SetTLSConfigFromFile("cert.pem", "key.pem")
	if e != nil {
		tp.Fatalf("%v", e)
	}

	// router
	srv.RouteCall(new(Math))

	// broadcast per 5s
	go func() {
		for {
			time.Sleep(time.Second * 5)
			srv.RangeSession(func(sess tp.Session) bool {
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
	tp.CallCtx
}

// Add handles addition request
func (m *Math) Add(arg *[]int) (int, *tp.Rerror) {
	// test query parameter
	tp.Infof("author: %s", m.PeekMeta("author"))
	// add
	var r int
	for _, a := range *arg {
		r += a
	}
	// response
	return r, nil
}
