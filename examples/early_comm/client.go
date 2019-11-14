package main

import (
	"github.com/henrylee2cn/erpc/v6"
)

//go:generate go build $GOFILE

func main() {
	defer erpc.FlushLogger()
	cli := erpc.NewPeer(
		erpc.PeerConfig{
			PrintDetail: false,
		},
		new(earlyCall),
	)
	defer cli.Close()
	_, stat := cli.Dial(":9090")
	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}
}

type earlyCall struct{}

func (e *earlyCall) Name() string {
	return "early_call"
}

func (e *earlyCall) PostDial(sess erpc.PreSession, isRedial bool) *erpc.Status {
	stat := sess.PreSend(
		erpc.TypeCall,
		"/early/ping",
		map[string]string{
			"author": "henrylee2cn",
		},
		nil,
	)
	if !stat.OK() {
		return stat
	}

	input := sess.PreReceive(func(header erpc.Header) interface{} {
		if header.ServiceMethod() == "/early/pong" {
			return new(string)
		}
		erpc.Panicf("Received an unexpected response: %s", header.ServiceMethod())
		return nil
	})
	stat = input.Status()
	if !stat.OK() {
		return stat
	}
	erpc.Infof("result: %v", input.String())
	return nil
}
