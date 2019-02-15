package main

import (
	"context"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

//go:generate go build $GOFILE

func main() {
	defer tp.SetLoggerLevel("INFO")()
	cli := tp.NewPeer(tp.PeerConfig{PrintDetail: true})
	sess, err := cli.Dial(":9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}

	var result string
	sess.Call("/test/ok", "test1", &result)
	tp.Infof("test sync1: %v", result)
	result = ""
	rerr := sess.Call("/test/timeout", "test2", &result).Rerror()
	tp.Infof("test sync2: server context timeout: %v", rerr)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)

	result = ""
	callCmd := sess.AsyncCall(
		"/test/timeout",
		"test3",
		&result,
		make(chan tp.CallCmd, 1),
		tp.WithContext(ctx),
	)
	select {
	case <-callCmd.Done():
		cancel()
		tp.Infof("test async1: %v", result)
	case <-ctx.Done():
		tp.Warnf("test async1: client context timeout: %v", ctx.Err())
	}

	time.Sleep(time.Second * 6)
	result = ""
	rerr = sess.Call("/test/ok", "test4", &result).Rerror()
	tp.Warnf("test sync3: disconnect due to server session timeout: %v", rerr.ToError())

	sess, err = cli.Dial(":9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}
	sess.AsyncCall(
		"/test/break",
		nil,
		nil,
		make(chan tp.CallCmd, 1),
	)
}
