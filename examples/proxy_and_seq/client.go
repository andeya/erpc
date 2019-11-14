package main

import (
	"fmt"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/socket"
)

//go:generate go build $GOFILE

func main() {
	defer erpc.SetLoggerLevel("ERROR")()

	cli := erpc.NewPeer(
		erpc.PeerConfig{},
	)
	defer cli.Close()

	sess, stat := cli.Dial(":8080")
	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}

	var result int
	stat = sess.Call("/math/add",
		[]int{1, 2, 3, 4, 5},
		&result,
	).Status()

	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}
	erpc.Printf("result: %d", result)

	stat = sess.Push(
		"/chat/say",
		fmt.Sprintf("I get result %d", result),
		socket.WithSetMeta("X-ID", "client-001"),
	)
	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}
}
