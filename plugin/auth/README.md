## auth

An auth plugin for verifying peer at the first time.


#### Test

```go
package auth_test

import (
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/plugin/auth"
)

func Test(t *testing.T) {
	// Server
	srv := tp.NewPeer(
		tp.PeerConfig{ListenPort: 9090},
		authChecker,
	)
	srv.RouteCall(new(Home))
	go srv.ListenAndServe()
	time.Sleep(1e9)

	// Client
	cli := tp.NewPeer(
		tp.PeerConfig{},
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
		tp.WithAddMeta("peer_id", "110"),
	).Status()
	if !stat.OK() {
		t.Error(stat)
	}
	t.Logf("result:%v", result)
	time.Sleep(3e9)
}

const clientAuthInfo = "client-auth-info-12345"

var authBearer = auth.NewBearerPlugin(
	func(sess auth.Session, fn auth.SendOnce) (stat *tp.Status) {
		var ret string
		stat = fn(clientAuthInfo, &ret)
		if !stat.OK() {
			return
		}
		tp.Infof("auth info: %s, result: %s", clientAuthInfo, ret)
		return
	},
	tp.WithBodyCodec('s'),
)

var authChecker = auth.NewCheckerPlugin(
	func(sess auth.Session, fn auth.RecvOnce) (ret interface{}, stat *tp.Status) {
		var authInfo string
		stat = fn(&authInfo)
		if !stat.OK() {
			return
		}
		tp.Infof("auth info: %v", authInfo)
		if clientAuthInfo != authInfo {
			return nil, tp.NewStatus(403, "auth fail", "auth fail detail")
		}
		return "pass", nil
	},
	tp.WithBodyCodec('s'),
)

type Home struct {
	tp.CallCtx
}

func (h *Home) Test(arg *map[string]string) (map[string]interface{}, *tp.Status) {
	return map[string]interface{}{
		"arg": *arg,
	}, nil
}
```

test command:

```sh
go test -v -run=Test
```