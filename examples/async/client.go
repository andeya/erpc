package main

import (
	"fmt"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	tp.SetLoggerLevel("INFO")
	cli := tp.NewPeer(tp.PeerConfig{})
	defer cli.Close()
	sess, err := cli.Dial(":9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}

	// Single asynchronous pull
	var result string
	pullCmd := sess.AsyncPull(
		"/test/wait3s",
		"Single asynchronous pull",
		&result,
		make(chan tp.PullCmd, 1),
	)
WAIT:
	for {
		select {
		case <-pullCmd.Done():
			tp.Infof("test 1: result: %#v, error: %v", result, pullCmd.Rerror())
			break WAIT
		default:
			tp.Warnf("test 1: Not yet returned to the result, try again later...")
			time.Sleep(time.Second)
		}
	}

	// Batch asynchronous pull
	batch := 10
	pullCmdChan := make(chan tp.PullCmd, batch)
	for i := 0; i < batch; i++ {
		sess.AsyncPull(
			"/test/wait3s",
			fmt.Sprintf("Batch asynchronous pull %d", i+1),
			new(string),
			pullCmdChan,
		)
	}
	for pullCmd := range pullCmdChan {
		result, rerr := pullCmd.Reply()
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
