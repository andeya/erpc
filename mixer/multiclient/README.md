## multiclient

Higher throughput client connection pool when transferring large messages (such as downloading files).

### Feature

- Non exclusive, shared connection pool
- Making full use of the asynchronous communication advantages of each connection
- Higher throughput when transferring large messages(≥1MB)
- Preempt more network bandwidth in a shared network environment
- Load balancing mechanism of traffic level
- Real-time monitoring of connection status

### Usage
	
`import "github.com/henrylee2cn/erpc/v6/mixer/multiclient"`

#### Test

```go
package multiclient_test

import (
	"testing"
	"time"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/mixer/multiclient"
)

type Arg struct {
	A int
	B int `param:"<range:1:>"`
}

type P struct{ erpc.CallCtx }

func (p *P) Divide(arg *Arg) (int, *erpc.Status) {
	return arg.A / arg.B, nil
}

func TestMultiClient(t *testing.T) {
	srv := erpc.NewPeer(erpc.PeerConfig{
		ListenPort: 9090,
	})
	srv.RouteCall(new(P))
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := multiclient.New(
		erpc.NewPeer(erpc.PeerConfig{}),
		":9090",
		100,
		time.Second*5,
	)
	go func() {
		for {
			t.Logf("%+v", cli.Stats())
			time.Sleep(time.Millisecond * 500)
		}
	}()
	go func() {
		var result int
		for i := 0; ; i++ {
			stat := cli.Call("/p/divide", &Arg{
				A: i,
				B: 2,
			}, &result).Status()
			if !stat.OK() {
				t.Log(stat)
			} else {
				t.Logf("%d/2=%v", i, result)
			}
			time.Sleep(time.Millisecond * 500)
		}
	}()
	time.Sleep(time.Second * 6)
	cli.Close()
	time.Sleep(time.Second * 3)
}
```

test command:

```sh
go test -v -run=TestMultiClient
```