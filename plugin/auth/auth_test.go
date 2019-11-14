package auth_test

import (
	"testing"
	"time"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/plugin/auth"
)

func Test(t *testing.T) {
	// Server
	srv := erpc.NewPeer(
		erpc.PeerConfig{ListenPort: 9090, CountTime: true},
		authChecker,
	)
	srv.RouteCall(new(Home))
	go srv.ListenAndServe()
	time.Sleep(1e9)

	// Client
	cli := erpc.NewPeer(
		erpc.PeerConfig{CountTime: true},
		authBearer,
	)
	sess, stat := cli.Dial(":9090")
	if !stat.OK() {
		t.Fatal(stat)
	}
	var result interface{}
	stat = sess.Call("/home/test",
		map[string]string{
			"author": "henrylee2cn",
		},
		&result,
		erpc.WithAddMeta("peer_id", "110"),
	).Status()
	if !stat.OK() {
		t.Error(stat)
	}
	t.Logf("result:%v", result)
	time.Sleep(3e9)
}

const clientAuthInfo = "client-auth-info-12345"

var authBearer = auth.NewBearerPlugin(
	func(sess auth.Session, fn auth.SendOnce) (stat *erpc.Status) {
		var ret string
		stat = fn(clientAuthInfo, &ret)
		if !stat.OK() {
			return
		}
		erpc.Infof("auth info: %s, result: %s", clientAuthInfo, ret)
		return
	},
	erpc.WithBodyCodec('s'),
)

var authChecker = auth.NewCheckerPlugin(
	func(sess auth.Session, fn auth.RecvOnce) (ret interface{}, stat *erpc.Status) {
		var authInfo string
		stat = fn(&authInfo)
		if !stat.OK() {
			return
		}
		erpc.Infof("auth info: %v", authInfo)
		if clientAuthInfo != authInfo {
			return nil, erpc.NewStatus(403, "auth fail", "auth fail detail")
		}
		return "pass", nil
	},
	erpc.WithBodyCodec('s'),
)

type Home struct {
	erpc.CallCtx
}

func (h *Home) Test(arg *map[string]string) (map[string]interface{}, *erpc.Status) {
	return map[string]interface{}{
		"arg": *arg,
	}, nil
}
