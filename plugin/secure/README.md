## secure

Package secure encrypting/decrypting the message body.

### Usage

`import "github.com/henrylee2cn/erpc/v6/plugin/secure"`

Ciphertext struct:

```go
package secure_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/plugin/secure"
)

type Arg struct {
	A int
	B int
}

type Result struct {
	C int
}

type math struct{ erpc.CallCtx }

func (m *math) Add(arg *Arg) (*Result, *erpc.Status) {
	// enforces the body of the encrypted reply message.
	// secure.EnforceSecure(m.Output())
	return &Result{C: arg.A + arg.B}, nil
}

func newSession(t *testing.T, port uint16) erpc.Session {
	p := secure.NewPlugin(100001, "cipherkey1234567")
	srv := erpc.NewPeer(erpc.PeerConfig{
		ListenPort:  port,
		PrintDetail: true,
	})
	srv.RouteCall(new(math), p)
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := erpc.NewPeer(erpc.PeerConfig{
		PrintDetail: true,
	}, p)
	sess, stat := cli.Dial(":" + strconv.Itoa(int(port)))
	if !stat.OK() {
		t.Fatal(stat)
	}
	return sess
}

func TestSecurePlugin(t *testing.T) {
	sess := newSession(t, 9090)
	// test secure
	var result Result
	stat := sess.Call(
		"/math/add",
		&Arg{A: 10, B: 2},
		&result,
		secure.WithSecureMeta(),
		// secure.WithAcceptSecureMeta(false),
	).Status()
	if !stat.OK() {
		t.Fatal(stat)
	}
	if result.C != 12 {
		t.Fatalf("expect 12, but get %d", result.C)
	}
	t.Logf("test secure10+2=%d", result.C)
}

func TestAcceptSecurePlugin(t *testing.T) {
	sess := newSession(t, 9091)
	// test accept secure
	var result Result
	stat := sess.Call(
		"/math/add",
		&Arg{A: 20, B: 4},
		&result,
		secure.WithAcceptSecureMeta(true),
	).Status()
	if !stat.OK() {
		t.Fatal(stat)
	}
	if result.C != 24 {
		t.Fatalf("expect 24, but get %d", result.C)
	}
	t.Logf("test accept secure: 20+4=%d", result.C)
}
```

test command:

```sh
go test -v -run=TestSecurePlugin
go test -v -run=TestAcceptSecurePlugin
```