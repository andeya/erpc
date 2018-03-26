package main

import (
	"fmt"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/socket"
)

func main() {
	tp.SetLoggerLevel("ERROR")

	cli := tp.NewPeer(
		tp.PeerConfig{},
	)
	defer cli.Close()

	sess, err := cli.Dial(":8080")
	if err != nil {
		tp.Fatalf("%v", err)
	}

	var reply int
	rerr := sess.Pull("/math/add",
		[]int{1, 2, 3, 4, 5},
		&reply,
	).Rerror()

	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
	tp.Printf("reply: %d", reply)

	rerr = sess.Push(
		"/chat/say",
		fmt.Sprintf("I get result %d", reply),
		socket.WithSetMeta("X-ID", "client-001"),
	)
	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
}
