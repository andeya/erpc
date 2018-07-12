package main

import (
	"context"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	tp.SetLoggerLevel("INFO")
	cli := tp.NewPeer(tp.PeerConfig{PrintDetail: true})
	sess, err := cli.Dial(":9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}

	var result string
	sess.Pull("/test/ok", "test1", &result)
	tp.Infof("test sync1: %v", result)
	result = ""
	rerr := sess.Pull("/test/timeout", "test2", &result).Rerror()
	tp.Infof("test sync2: server context timeout: %v", rerr)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)

	result = ""
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
		tp.Infof("test async1: %v", result)
	case <-ctx.Done():
		tp.Warnf("test async1: client context timeout: %v", ctx.Err())
	}

	time.Sleep(time.Second * 6)
	result = ""
	rerr = sess.Pull("/test/ok", "test4", &result).Rerror()
	tp.Warnf("test sync3: disconnect due to server session timeout: %v", rerr.ToError())

	sess, err = cli.Dial(":9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}
	sess.AsyncPull(
		"/test/break",
		nil,
		nil,
		make(chan tp.PullCmd, 1),
	)
}
