## tps

tps statistics requests per second.

### Usage

`import tps "github.com/henrylee2cn/tp-ext/plugin-tps"`

#### Test

```go
package tps

import (
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

type Call struct {
	tp.CallCtx
}

func (*Call) Test(*struct{}) (*struct{}, *tp.Rerror) {
	return nil, nil
}

type Push struct {
	tp.PushCtx
}

func (*Push) Test(*struct{}) *tp.Rerror {
	return nil
}

func TestTPS(t *testing.T) {
	tp.SetLoggerLevel("OFF")
	// Server
	srv := tp.NewPeer(tp.PeerConfig{ListenPort: 9090}, NewTPS(5))
	srv.RouteCall(new(Call))
	srv.RoutePush(new(Push))
	go srv.ListenAndServe()
	time.Sleep(1e9)

	// Client
	cli := tp.NewPeer(tp.PeerConfig{})
	sess, err := cli.Dial(":9090")
	if err != nil {
		if err != nil {
			t.Error(err)
		}
	}
	var (
		rerr   *tp.Rerror
		ticker = time.NewTicker(time.Millisecond * 10)
	)
	for i := 0; i < 1<<10; i++ {
		<-ticker.C
		rerr = sess.Call("/call/test", nil, nil).Rerror()
		if rerr != nil {
			t.Error(rerr)
		}
		rerr = sess.Push("/push/test", nil)
		if rerr != nil {
			t.Error(rerr)
		}
	}
}
```

test command:

```sh
go test -v -run=TestTPS
```
