package proxy_and_seq

import (
	"fmt"
	"testing"

	"github.com/andeya/erpc/v7"
	"github.com/andeya/erpc/v7/socket"
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

	sess, stat := cli.Dial(":8080")
	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}

	var result int
	stat = sess.Call("/math/add",
		[]int{1, 2, 3, 4, 5},
		&result,
	).Status()

	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}
	erpc.Printf("result: %d", result)

	stat = sess.Push(
		"/chat/say",
		fmt.Sprintf("I get result %d", result),
		socket.WithSetMeta("X-ID", "client-001"),
	)
	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}
}
