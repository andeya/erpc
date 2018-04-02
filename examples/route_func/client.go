package main

import (
	tp "github.com/henrylee2cn/teleport"
)

func main() {
	tp.SetLoggerLevel("ERROR")

	cli := tp.NewPeer(tp.PeerConfig{})
	defer cli.Close()

	cli.RoutePushFunc((*ctrl).ServerStatus1)
	cli.RoutePushFunc(ServerStatus2)

	sess, err := cli.Dial(":9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}

	var reply int
	rerr := sess.Pull("/math/add1?push_status=yes",
		[]int{1, 2, 3, 4, 5},
		&reply,
	).Rerror()

	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
	tp.Printf("reply1: %d", reply)

	rerr = sess.Pull("/math/add2?push_status=yes",
		[]int{1, 2, 3, 4, 5},
		&reply,
	).Rerror()

	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
	tp.Printf("reply2: %d", reply)
}

type ctrl struct {
	tp.PushCtx
}

func (c *ctrl) ServerStatus1(args *string) *tp.Rerror {
	return ServerStatus2(c, args)
}

func ServerStatus2(ctx tp.PushCtx, args *string) *tp.Rerror {
	tp.Printf("server status(%s): %s", ctx.Uri(), *args)
	return nil
}
