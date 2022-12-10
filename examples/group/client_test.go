package group

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

	cli := erpc.NewPeer(
		erpc.PeerConfig{},
	)
	defer cli.Close()
	group := cli.SubRoute("/cli")
	group.RoutePush(new(push))

	sess, stat := cli.Dial(":9090")
	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}

	var result int
	stat = sess.Call("/srv/math/v2/add_2",
		[]int{1, 2, 3, 4, 5},
		&result,
		erpc.WithSetMeta("push_status", "yes"),
	).Status()

	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}
	erpc.Printf("result: %d", result)
}

type push struct {
	erpc.PushCtx
}

func (p *push) ServerStatus(arg *string) *erpc.Status {
	erpc.Printf("server status: %s", *arg)
	return nil
}
