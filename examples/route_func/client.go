package main

import (
	tp "github.com/henrylee2cn/teleport/v6"
)

//go:generate go build $GOFILE

func main() {
	defer tp.SetLoggerLevel("ERROR")()

	cli := tp.NewPeer(tp.PeerConfig{})
	defer cli.Close()

	cli.RoutePushFunc((*ctrl).ServerStatus1)
	cli.RoutePushFunc(ServerStatus2)

	sess, stat := cli.Dial(":9090")
	if !stat.OK() {
		tp.Fatalf("%v", stat)
	}

	var result int
	stat = sess.Call("/math/add1",
		[]int{1, 2, 3, 4, 5},
		&result,
	).Status()

	if !stat.OK() {
		tp.Fatalf("%v", stat)
	}
	tp.Printf("result1: %d", result)

	stat = sess.Call("/math/add2",
		[]int{1, 2, 3, 4, 5},
		&result,
		tp.WithAddMeta("push_status", "yes"),
	).Status()

	if !stat.OK() {
		tp.Fatalf("%v", stat)
	}
	tp.Printf("result2: %d", result)
}

type ctrl struct {
	tp.PushCtx
}

func (c *ctrl) ServerStatus1(arg *string) *tp.Status {
	return ServerStatus2(c, arg)
}

func ServerStatus2(ctx tp.PushCtx, arg *string) *tp.Status {
	tp.Printf("server status(%s): %s", ctx.ServiceMethod(), *arg)
	return nil
}
