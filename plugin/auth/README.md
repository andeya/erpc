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
	sess, rerr := cli.Dial(":9090")
	if rerr.HasError() {
		t.Fatal(rerr)
	}
	var result interface{}
	rerr = sess.Call("/home/test",
		map[string]string{
			"author": "henrylee2cn",
		},
		&result,
		tp.WithAddMeta("peer_id", "110"),
	).Rerror()
	if rerr != nil {
		t.Error(rerr)
	}
	t.Logf("result:%v", result)
	time.Sleep(3e9)
}

const clientAuthInfo = "client-auth-info-12345"

var authBearer = auth.NewBearerPlugin(
	func(sess auth.Session, fn auth.Sender) *tp.Rerror {
		var ret string
		rerr := fn(clientAuthInfo, &ret)
		if rerr.HasError() {
			tp.Infof("auth info: %s, error: %s", clientAuthInfo, rerr)
			return rerr
		}
		tp.Infof("auth info: %s, result: %s", clientAuthInfo, ret)
		return nil
	},
	tp.WithBodyCodec('s'),
)

var authChecker = auth.NewCheckerPlugin(
	func(sess auth.Session, fn auth.Receiver) (ret interface{}, rerr *tp.Rerror) {
		var authInfo string
		rerr = fn(&authInfo)
		if rerr.HasError() {
			return
		}
		tp.Infof("auth info: %v", authInfo)
		if clientAuthInfo != authInfo {
			return nil, tp.NewRerror(403, "auth fail", "auth fail detail")
		}
		return "pass", nil
	},
	tp.WithBodyCodec('s'),
)

type Home struct {
	tp.CallCtx
}

func (h *Home) Test(arg *map[string]string) (map[string]interface{}, *tp.Rerror) {
	return map[string]interface{}{
		"arg": *arg,
	}, nil
}
```

test command:

```sh
go test -v -run=Test
```