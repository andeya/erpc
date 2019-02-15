package main

import (
	"time"

	tp "github.com/henrylee2cn/teleport"
)

//go:generate go build $GOFILE

func main() {
	defer tp.SetLoggerLevel("ERROR")()

	cli := tp.NewPeer(tp.PeerConfig{})
	defer cli.Close()
	// cli.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})

	cli.RoutePush(new(Push))

	sess, err := cli.Dial(":9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}

	var result int
	rerr := sess.Call("/math/add",
		[]int{1, 2, 3, 4, 5},
		&result,
		tp.WithAddMeta("author", "henrylee2cn"),
	).Rerror()
	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
	tp.Printf("result: %d", result)

	tp.Printf("wait for 10s...")
	time.Sleep(time.Second * 10)
}

// Push push handler
type Push struct {
	tp.PushCtx
}

// Push handles '/push/status' message
func (p *Push) Status(arg *string) *tp.Rerror {
	tp.Printf("%s", *arg)
	return nil
}
