package main

import (
	tp "github.com/henrylee2cn/teleport"
)

//go:generate go build $GOFILE

func main() {
	defer tp.SetLoggerLevel("ERROR")()

	cli := tp.NewPeer(
		tp.PeerConfig{},
	)
	defer cli.Close()
	group := cli.SubRoute("/cli")
	group.RoutePush(new(push))

	sess, err := cli.Dial(":9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}

	var result int
	rerr := sess.Call("/srv/math/v2/add_2",
		[]int{1, 2, 3, 4, 5},
		&result,
		tp.WithSetMeta("push_status", "yes"),
	).Rerror()

	if rerr != nil {
		tp.Fatalf("%v", err)
	}
	tp.Printf("result: %d", result)
}

type push struct {
	tp.PushCtx
}

func (p *push) ServerStatus(arg *string) *tp.Rerror {
	tp.Printf("server status: %s", *arg)
	return nil
}
