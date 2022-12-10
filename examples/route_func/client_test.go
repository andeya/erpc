package route_func

import (
	"testing"

	"github.com/andeya/erpc/v7"
	"github.com/andeya/goutil"
)

//go:generate go test -v -c -o "${GOPACKAGE}_client" $GOFILE

func TestClient(t *testing.T) {
	if goutil.IsGoTest() {
		t.Log("skip test in go test")
		return
	}
	defer erpc.SetLoggerLevel("ERROR")()

	cli := erpc.NewPeer(erpc.PeerConfig{})
	defer cli.Close()

	cli.RoutePushFunc((*pushCtrl).ServerStatus1)
	cli.RoutePushFunc(ServerStatus2)

	sess, stat := cli.Dial(":9090")
	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}

	var result int
	stat = sess.Call("/math/add1",
		[]int{1, 2, 3, 4, 5},
		&result,
	).Status()

	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}
	erpc.Printf("result1: %d", result)

	stat = sess.Call("/math/add2",
		[]int{1, 2, 3, 4, 5},
		&result,
		erpc.WithAddMeta("push_status", "yes"),
	).Status()

	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}
	erpc.Printf("result2: %d", result)
}

type pushCtrl struct {
	erpc.PushCtx
}

func (c *pushCtrl) ServerStatus1(arg *string) *erpc.Status {
	return ServerStatus2(c, arg)
}

func ServerStatus2(ctx erpc.PushCtx, arg *string) *erpc.Status {
	erpc.Printf("server status(%s): %s", ctx.ServiceMethod(), *arg)
	return nil
}
