package main

import (
	"fmt"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/socket"
)

//go:generate go build $GOFILE

func main() {
	defer tp.SetLoggerLevel("ERROR")()

	cli := tp.NewPeer(
		tp.PeerConfig{},
	)
	defer cli.Close()

	sess, err := cli.Dial(":8080")
	if err != nil {
		tp.Fatalf("%v", err)
	}

	var result int
	rerr := sess.Call("/math/add",
		[]int{1, 2, 3, 4, 5},
		&result,
	).Rerror()

	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
	tp.Printf("result: %d", result)

	rerr = sess.Push(
		"/chat/say",
		fmt.Sprintf("I get result %d", result),
		socket.WithSetMeta("X-ID", "client-001"),
	)
	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
}
