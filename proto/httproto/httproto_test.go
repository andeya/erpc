package httproto_test

import (
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/henrylee2cn/goutil/httpbody"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/proto/httproto"
	"github.com/henrylee2cn/teleport/xfer/gzip"
)

type Home struct {
	tp.CallCtx
}

func (h *Home) Test(arg *map[string]string) (map[string]interface{}, *tp.Rerror) {
	tp.Infof("peer_id: %s", h.PeekMeta("peer_id"))
	return map[string]interface{}{
		"arg": *arg,
	}, nil
}

func TestHTTProto(t *testing.T) {
	gzip.Reg('g', "gizp-5", 5)

	// Server
	srv := tp.NewPeer(tp.PeerConfig{ListenPort: 9090})
	srv.RouteCall(new(Home))
	go srv.ListenAndServe(httproto.NewHTTProtoFunc(true))
	time.Sleep(1e9)

	url := "http://localhost:9090/home/test?peer_id=110"
	// TP Client
	cli := tp.NewPeer(tp.PeerConfig{})
	sess, rerr := cli.Dial(":9090", httproto.NewHTTProtoFunc())
	if rerr != nil {
		t.Fatal(rerr)
	}
	var result interface{}
	var arg = map[string]string{
		"author": "henrylee2cn",
	}
	rerr = sess.Call(
		url,
		arg,
		&result,
		// tp.WithXferPipe('g'),
	).Rerror()
	if rerr != nil {
		t.Fatal(rerr)
	}
	t.Logf("teleport client response: %v", result)

	// HTTP Client
	contentType, body, _ := httpbody.NewJSONBody(arg)
	resp, err := http.Post(url, contentType, body)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	t.Logf("http client response: %s", b)
}
