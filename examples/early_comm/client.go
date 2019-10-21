package main

import (
	tp "github.com/henrylee2cn/teleport"
)

//go:generate go build $GOFILE

func main() {
	defer tp.FlushLogger()
	cli := tp.NewPeer(
		tp.PeerConfig{
			PrintDetail: false,
		},
		new(earlyCall),
	)
	defer cli.Close()
	_, stat := cli.Dial(":9090")
	if !stat.OK() {
		tp.Fatalf("%v", stat)
	}
}

type earlyCall struct{}

func (e *earlyCall) Name() string {
	return "early_call"
}

func (e *earlyCall) PostDial(sess tp.PreSession, isRedial bool) *tp.Status {
	stat := sess.PreSend(
		tp.TypeCall,
		"/early/ping",
		map[string]string{
			"author": "henrylee2cn",
		},
		nil,
	)
	if !stat.OK() {
		return stat
	}

	input := sess.PreReceive(func(header tp.Header) interface{} {
		if header.ServiceMethod() == "/early/pong" {
			return new(string)
		}
		tp.Panicf("Received an unexpected response: %s", header.ServiceMethod())
		return nil
	})
	stat = input.Status()
	if !stat.OK() {
		return stat
	}
	tp.Infof("result: %v", input.String())
	return nil
}
