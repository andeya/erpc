package main

import (
	"time"

	tp "github.com/henrylee2cn/teleport/v6"
)

//go:generate go build $GOFILE

func main() {
	defer tp.SetLoggerLevel("ERROR")()

	cli := tp.NewPeer(tp.PeerConfig{Network: "quic"})
	defer cli.Close()
	e := cli.SetTLSConfigFromFile("cert.pem", "key.pem", true)
	if e != nil {
		tp.Fatalf("%v", e)
	}

	cli.RoutePush(new(Push))

	sess, stat := cli.Dial(":9090")
	if !stat.OK() {
		tp.Fatalf("%v", stat)
	}

	var result int
	stat = sess.Call("/math/add",
		[]int{1, 2, 3, 4, 5},
		&result,
		tp.WithAddMeta("author", "henrylee2cn"),
	).Status()
	if !stat.OK() {
		tp.Fatalf("%v", stat)
	}
	tp.Printf("result: %d", result)

	tp.Printf("wait for 10s...")
	time.Sleep(time.Second * 10)
}

// Push push handler
type Push struct {
	tp.PushCtx
}

// Push handles '/push/status' message
func (p *Push) Status(arg *string) *tp.Status {
	tp.Printf("%s", *arg)
	return nil
}
