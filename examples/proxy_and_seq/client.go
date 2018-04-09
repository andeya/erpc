package main

import (
	"fmt"

	"github.com/henrylee2cn/goutil"
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
		tp.WithSeq(newRequestId()),
	).Rerror()

	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
	tp.Printf("reply: %d", reply)

	rerr = sess.Push(
		"/chat/say",
		fmt.Sprintf("I get result %d", reply),
		socket.WithSetMeta("X-ID", "client-001"),
		tp.WithSeq(newRequestId()),
	)
	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
}

func newRequestId() string {
	return "uid@" + goutil.URLRandomString(8)
}
