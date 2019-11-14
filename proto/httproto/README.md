## httproto

httproto is implemented HTTP style socket communication protocol.

NOTE: It simply transfers data in HTTP style instead of the full HTTP protocol.

### Message

example:

- Request Message

```
POST /home/test?peer_id=110 HTTP/1.1
Accept-Encoding: gzip
Content-Length: 24
Content-Type: application/json;charset=utf-8
Host: localhost:9090
User-Agent: erpc-httproto/1.1
X-Mtype: 1
X-Seq: 1

{"author":"henrylee2cn"}
```

- Response Message

```
HTTP/1.1 200 OK
Content-Length: 32
Content-Type: application/json;charset=utf-8
X-Mtype: 2
X-Seq: 1

{"arg":{"author":"henrylee2cn"}}
```

or

```
HTTP/1.1 299 Business Error
Content-Length: 56
Content-Type: application/json
X-Mtype: 2
X-Seq: 0

{"code":1,"msg":"test error","cause":"this is test:110"}
```

- Default Support Content-Type
	- codec.ID_PROTOBUF: application/x-protobuf;charset=utf-8
	- codec.ID_JSON:     application/json;charset=utf-8
	- codec.ID_FORM:     application/x-www-form-urlencoded;charset=utf-8
	- codec.ID_PLAIN:    text/plain;charset=utf-8
	- codec.ID_XML:      text/xml;charset=utf-8


- RegBodyCodec registers a mapping of content type to body coder

```go
func RegBodyCodec(contentType string, codecID byte)
```

### Usage

`import "github.com/henrylee2cn/erpc/v6/proto/httproto"`

#### Test

```go
package httproto_test

import (
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/henrylee2cn/goutil/httpbody"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/proto/httproto"
	"github.com/henrylee2cn/erpc/v6/xfer/gzip"
)

type Home struct {
	erpc.CallCtx
}

func (h *Home) Test(arg *map[string]string) (map[string]interface{}, *erpc.Status) {
	erpc.Infof("peer_id: %s", h.PeekMeta("peer_id"))
	return map[string]interface{}{
		"arg": *arg,
	}, nil
}

func TestHTTProto(t *testing.T) {
	gzip.Reg('g', "gizp-5", 5)

	// Server
	srv := erpc.NewPeer(erpc.PeerConfig{ListenPort: 9090})
	srv.RouteCall(new(Home))
	go srv.ListenAndServe(httproto.NewHTTProtoFunc(true))
	time.Sleep(1e9)

	url := "http://localhost:9090/home/test?peer_id=110"
	// TP Client
	cli := erpc.NewPeer(erpc.PeerConfig{})
	sess, stat := cli.Dial(":9090", httproto.NewHTTProtoFunc())
	if !stat.OK() {
		t.Fatal(stat)
	}
	var result interface{}
	var arg = map[string]string{
		"author": "henrylee2cn",
	}
	stat = sess.Call(
		url,
		arg,
		&result,
		// erpc.WithXferPipe('g'),
	).Status()
	if !stat.OK() {
		t.Fatal(stat)
	}
	t.Logf("erpc client response: %v", result)

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
```

test command:

```sh
go test -v -run=TestHTTProto
```
