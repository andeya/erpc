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

	var result string
	sess.Pull("/test/ok", "test1", &result)
	tp.Infof("test sync 1: %v", result)

	sess.Pull("/test/timeout", "test2", &result).Rerror()
	tp.Infof("test sync 2: server context timeout: %v", result)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)

	pullCmd := sess.AsyncPull(
		"/test/timeout",
		"test3",
		&result,
		make(chan tp.PullCmd, 1),
		tp.WithContext(ctx),
	)
	select {
	case <-pullCmd.Done():
		cancel()
		tp.Infof("test async: %v", result)
	case <-ctx.Done():
		tp.Warnf("test async: client context timeout: %v", ctx.Err())
	}

	time.Sleep(time.Second * 5)
	rerr := sess.Pull("/test/ok", "test4", &result).Rerror()
	tp.Warnf("test sync 3: disconnect due to server session timeout: %v", rerr.ToError())
}
