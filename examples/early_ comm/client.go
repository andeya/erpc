package main

import (
	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/socket"
)

func main() {
	cli := tp.NewPeer(
		tp.PeerConfig{
			PrintDetail: false,
		},
		new(earlyPull),
	)
	defer cli.Close()
	_, err := cli.Dial(":9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}
}

type earlyPull struct{}

func (e *earlyPull) Name() string {
	return "early_pull"
}

func (e *earlyPull) PostDial(sess tp.PreSession) *tp.Rerror {
	rerr := sess.Send(
		"/early/ping",
		map[string]string{
			"author": "henrylee2cn",
		},
		nil,
	)
	if rerr != nil {
		return rerr
	}

	input, rerr := sess.Receive(func(header socket.Header) interface{} {
		if header.Uri() == "/early/pong" {
			return new(string)
		}
		tp.Panicf("Received an unexpected response: %s", header.Uri())
		return nil
	})
	if rerr != nil {
		return rerr
	}
	tp.Infof("result: %v", input.String())
	return nil
}
