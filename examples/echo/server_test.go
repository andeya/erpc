package echo

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/andeya/erpc/v7"
	"github.com/andeya/goutil"
)

//go:generate go test -v -c -o "${GOPACKAGE}_server" $GOFILE

func TestServer(t *testing.T) {
	if goutil.IsGoTest() {
		t.Log("skip test in go test")
		return
	}

	defer erpc.FlushLogger()
	srv := erpc.NewPeer(erpc.PeerConfig{
		CountTime:  true,
		ListenPort: 9090,
	})
	srv.RouteCall(func() erpc.CtrlStructPtr { return &Echo{Suffix: strconv.FormatInt(time.Now().UnixMilli(), 10)} })
	srv.ListenAndServe()
}

type Echo struct {
	erpc.CallCtx
	Suffix string
}

func (echo *Echo) AddSuffix(arg *string) (string, *erpc.Status) {
	return fmt.Sprintf("%s ------ %s", *arg, echo.Suffix), nil
}
