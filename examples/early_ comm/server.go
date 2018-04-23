package main

import (
	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/socket"
)

func main() {
	srv := tp.NewPeer(
		tp.PeerConfig{
			PrintDetail:   false,
			ListenAddress: ":9090",
		},
		new(earlyReply),
	)
	srv.ListenAndServe()
}

type earlyReply struct{}

func (e *earlyReply) Name() string {
	return "early_reply"
}

func (e *earlyReply) PostAccept(sess tp.PreSession) *tp.Rerror {
	var rigthUri bool
	input, rerr := sess.Receive(func(header socket.Header) interface{} {
		if header.Uri() == "/early/ping" {
			rigthUri = true
			return new(map[string]string)
		}
		return nil
	})
	if rerr != nil {
		return rerr
	}

	var reply string
	if !rigthUri {
		rerr = tp.NewRerror(10005, "unexpected request", "")
	} else {
		body := *input.Body().(*map[string]string)
		if body["author"] != "henrylee2cn" {
			rerr = tp.NewRerror(10005, "incorrect author", body["author"])
		} else {
			rerr = nil
			reply = "OK"
		}
	}
	return sess.Send(
		"/early/pong",
		reply,
		rerr,
	)
}
