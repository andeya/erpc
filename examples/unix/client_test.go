package unix

import (
	"net"
	"os"
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
		erpc.PeerConfig{
			Network:   "unix",
			LocalPort: 9091,
		},
		&erpc.PluginImpl{
			PluginName: "clean-local-unix-file",
			OnPreDial: func(localAddr net.Addr, remoteAddr string) (stat *erpc.Status) {
				addr := localAddr.String()
				if _, err := os.Stat(addr); err == nil {
					if err := os.RemoveAll(addr); err != nil {
						return erpc.NewStatusByCodeText(erpc.CodeDialFailed, err, false)
					}
				}
				return nil
			},
		},
	)
	defer cli.Close()

	sess, stat := cli.Dial("0.0.0.0:9090")
	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}

	var result string
	stat = sess.Call(
		"/echo/parrot",
		"this is request",
		&result,
	).Status()

	if !stat.OK() {
		erpc.Fatalf("%v", stat)
	}
	erpc.Printf("result: %s", result)
}
