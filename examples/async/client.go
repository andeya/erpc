package main

import (
	"fmt"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

//go:generate go build $GOFILE

func main() {
	defer tp.SetLoggerLevel("INFO")()
	cli := tp.NewPeer(tp.PeerConfig{})
	defer cli.Close()
	sess, err := cli.Dial(":9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}

	// Single asynchronous call
	var result string
	callCmd := sess.AsyncCall(
		"/test/wait3s",
		"Single asynchronous call",
		&result,
		make(chan tp.CallCmd, 1),
	)
WAIT:
	for {
		select {
		case <-callCmd.Done():
			tp.Infof("test 1: result: %#v, error: %v", result, callCmd.Rerror())
			break WAIT
		default:
			tp.Warnf("test 1: Not yet returned to the result, try again later...")
			time.Sleep(time.Second)
		}
	}

	// Batch asynchronous call
	batch := 10
	callCmdChan := make(chan tp.CallCmd, batch)
	for i := 0; i < batch; i++ {
		sess.AsyncCall(
			"/test/wait3s",
			fmt.Sprintf("Batch asynchronous call %d", i+1),
			new(string),
			callCmdChan,
		)
	}
	for callCmd := range callCmdChan {
		result, rerr := callCmd.Reply()
		if rerr != nil {
			tp.Errorf("test 2: error: %v", rerr)
		} else {
			tp.Infof("test 2: result: %v", *result.(*string))
		}
		batch--
		if batch == 0 {
			break
		}
	}
}
