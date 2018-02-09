package main

import (
	"context"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	tp.SetLoggerLevel("ERROR")
	cli := tp.NewPeer(tp.PeerConfig{})
	defer cli.Close()
	sess, err := cli.Dial(":9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}

	var reply string
	sess.Pull("/test/ok", "test1", &reply)
	tp.Printf("test normal: %v", reply)

	sess.Pull("/test/timeout", "test2", &reply).Rerror()
	tp.Printf("test: server context timeout: %v", reply)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	ch := make(chan tp.PullCmd, 1)
	sess.AsyncPull(
		"/test/timeout",
		"test3",
		&reply,
		ch,
		tp.WithContext(ctx),
	)
	select {
	case <-ch:
		cancel()
	case <-ctx.Done():
		tp.Printf("test: client context timeout: %v", ctx.Err())
	}

	time.Sleep(time.Second * 5)
	rerr := sess.Pull("/test/ok", "test4", &reply).Rerror()
	tp.Printf("test: disconnect due to server session timeout: %v", rerr.ToError())
}
