package main

import (
	"context"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	tp.SetLoggerLevel("INFO")
	cli := tp.NewPeer(tp.PeerConfig{PrintDetail: true})
	defer cli.Close()
	sess, err := cli.Dial(":9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}

	var reply string
	sess.Pull("/test/ok", "test1", &reply)
	tp.Infof("test sync 1: %v", reply)

	sess.Pull("/test/timeout", "test2", &reply).Rerror()
	tp.Infof("test sync 2: server context timeout: %v", reply)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)

	pullCmd := sess.AsyncPull(
		"/test/timeout",
		"test3",
		&reply,
		make(chan tp.PullCmd, 1),
		tp.WithContext(ctx),
	)
	select {
	case <-pullCmd.Done():
		cancel()
		tp.Infof("test async: %v", reply)
	case <-ctx.Done():
		tp.Warnf("test async: client context timeout: %v", ctx.Err())
	}

	time.Sleep(time.Second * 5)
	rerr := sess.Pull("/test/ok", "test4", &reply).Rerror()
	tp.Warnf("test sync 3: disconnect due to server session timeout: %v", rerr.ToError())
}
