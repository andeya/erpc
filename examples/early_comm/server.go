package main

import (
	"github.com/henrylee2cn/erpc/v6"
)

//go:generate go build $GOFILE

func main() {
	defer erpc.FlushLogger()
	srv := erpc.NewPeer(
		erpc.PeerConfig{
			PrintDetail: false,
			ListenPort:  9090,
		},
		new(earlyResult),
	)
	srv.ListenAndServe()
}

type earlyResult struct{}

func (e *earlyResult) Name() string {
	return "early_result"
}

func (e *earlyResult) PostAccept(sess erpc.PreSession) *erpc.Status {
	var rigthServiceMethod bool
	input := sess.PreReceive(func(header erpc.Header) interface{} {
		if header.ServiceMethod() == "/early/ping" {
			rigthServiceMethod = true
			return new(map[string]string)
		}
		return nil
	})
	stat := input.Status()
	if !stat.OK() {
		return stat
	}

	var result string
	if !rigthServiceMethod {
		stat = erpc.NewStatus(10005, "unexpected request", "")
	} else {
		body := *input.Body().(*map[string]string)
		if body["author"] != "henrylee2cn" {
			stat = erpc.NewStatus(10005, "incorrect author", body["author"])
		} else {
			stat = nil
			result = "OK"
		}
	}
	return sess.PreSend(
		erpc.TypeReply,
		"/early/pong",
		result,
		stat,
	)
}
