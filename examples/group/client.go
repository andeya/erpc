package main

import (
	tp "github.com/henrylee2cn/teleport"
)

func main() {
	tp.SetLoggerLevel("ERROR")

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

	var reply int
	rerr := sess.Pull("/srv/math/v2/add_2?push_status=yes",
		[]int{1, 2, 3, 4, 5},
		&reply,
	).Rerror()

	if rerr != nil {
		tp.Fatalf("%v", err)
	}
	tp.Printf("reply: %d", reply)
}

type push struct {
	tp.PushCtx
}

func (p *push) ServerStatus(args *string) *tp.Rerror {
	tp.Printf("server status: %s", *args)
	return nil
}
