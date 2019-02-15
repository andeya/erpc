package main

import (
	tp "github.com/henrylee2cn/teleport"
)

//go:generate go build $GOFILE

func main() {
	defer tp.SetLoggerLevel("ERROR")()

	cli := tp.NewPeer(tp.PeerConfig{})
	defer cli.Close()

	cli.RoutePushFunc((*ctrl).ServerStatus1)
	cli.RoutePushFunc(ServerStatus2)

	sess, err := cli.Dial(":9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}

	var result int
	rerr := sess.Call("/math/add1",
		[]int{1, 2, 3, 4, 5},
		&result,
	).Rerror()

	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
	tp.Printf("result1: %d", result)

	rerr = sess.Call("/math/add2",
		[]int{1, 2, 3, 4, 5},
		&result,
		tp.WithAddMeta("push_status", "yes"),
	).Rerror()

	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
	tp.Printf("result2: %d", result)
}

type ctrl struct {
	tp.PushCtx
}

func (c *ctrl) ServerStatus1(arg *string) *tp.Rerror {
	return ServerStatus2(c, arg)
}

func ServerStatus2(ctx tp.PushCtx, arg *string) *tp.Rerror {
	tp.Printf("server status(%s): %s", ctx.ServiceMethod(), *arg)
	return nil
}
