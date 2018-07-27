package main

import (
	tp "github.com/henrylee2cn/teleport"
)

func main() {
	tp.SetLoggerLevel("ERROR")
	cli := tp.NewPeer(tp.PeerConfig{})
	// cli := tp.NewPeer(tp.PeerConfig{LocalIP: "0.0.0.0"})
	defer cli.Close()
	cli.RoutePush(new(push))
	sess, err := cli.Dial(":9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}

	var result int
	rerr := sess.Call("/math/add?push_status=yes",
		[]int{1, 2, 3, 4, 5},
		&result,
		// tp.WithAcceptBodyCodec('s'),
	).Rerror()

	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
	tp.Printf("result: %d", result)
}

type push struct {
	tp.PushCtx
}

func (p *push) Status(arg *string) *tp.Rerror {
	tp.Printf("server status: %s", *arg)
	return nil
}
