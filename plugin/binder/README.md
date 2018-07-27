## binder

Parameter Binding Verification Plugin for Struct Handler.

### Usage

`import "github.com/henrylee2cn/teleport/plugin/binder"`

#### Param-Tags

tag   |   key    | required |     value     |   desc
------|----------|----------|---------------|----------------------------------
param |   query    | no |  (name e.g.`id`)   | It indicates that the parameter is from the URI query part. e.g. `/a/b?x={query}`
param |   swap    | no |   (name e.g.`id`)  | It indicates that the parameter is from the context swap.
param |   desc   |      no      |     (e.g.`id`)   | Parameter Description
param |   len    |      no      |   (e.g.`3:6`)  | Length range [a,b] of parameter's value
param |   range  |      no      |   (e.g.`0:10`)   | Numerical range [a,b] of parameter's value
param |  nonzero |      no      |    -    | Not allowed to zero
param |  regexp  |      no      |   (e.g.`^\w+$`)  | Regular expression validation
param |   rerr   |      no      |(e.g.`100002:wrong password format`)| Custom error code and message

NOTES:

* `param:"-"` means ignore
* Encountered untagged exportable anonymous structure field, automatic recursive resolution
* Parameter name is the name of the structure field converted to snake format
* If the parameter is not from `query` or `swap`, it is the default from the body

#### Field-Types

base    |   slice    | special
--------|------------|------------
string  |  []string  | [][]byte
byte    |  []byte    | [][]uint8
uint8   |  []uint8   | struct
bool    |  []bool    |
int     |  []int     |
int8    |  []int8    |
int16   |  []int16   |
int32   |  []int32   |
int64   |  []int64   |
uint8   |  []uint8   |
uint16  |  []uint16  |
uint32  |  []uint32  |
uint64  |  []uint64  |
float32 |  []float32 |
float64 |  []float64 |


#### Test

```go
package binder_test

import (
	"testing"
	"time"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/plugin/binder"
)

type (
	Arg struct {
		A int
		B int `param:"<range:1:100>"`
		Query
		XyZ       string  `param:"<query><nonzero><rerr: 100002: Parameter cannot be empty>"`
		SwapValue float32 `param:"<swap><nonzero>"`
	}
	Query struct {
		X string `param:"<query:_x>"`
	}
)

type P struct{ tp.CallCtx }

func (p *P) Divide(arg *Arg) (int, *tp.Rerror) {
	tp.Infof("query arg _x: %s, xy_z: %s, swap_value: %v", arg.Query.X, arg.XyZ, arg.SwapValue)
	return arg.A / arg.B, nil
}

type SwapPlugin struct{}

func (s *SwapPlugin) Name() string {
	return "swap_plugin"
}
func (s *SwapPlugin) PostReadCallBody(ctx tp.ReadCtx) *tp.Rerror {
	ctx.Swap().Store("swap_value", 123)
	return nil
}

func TestBinder(t *testing.T) {
	bplugin := binder.NewStructArgsBinder(nil)
	srv := tp.NewPeer(
		tp.PeerConfig{ListenPort: 9090},
	)
	srv.PluginContainer().AppendRight(bplugin)
	srv.RouteCall(new(P), new(SwapPlugin))
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := tp.NewPeer(tp.PeerConfig{})
	sess, err := cli.Dial(":9090")
	if err != nil {
		t.Fatal(err)
	}
	var result int
	rerr := sess.Call("/p/divide?_x=testquery_x&xy_z=testquery_xy_z", &Arg{
		A: 10,
		B: 2,
	}, &result).Rerror()
	if rerr != nil {
		t.Fatal(rerr)
	}
	t.Logf("10/2=%d", result)
	rerr = sess.Call("/p/divide", &Arg{
		A: 10,
		B: 5,
	}, &result).Rerror()
	if rerr == nil {
		t.Fatal(rerr)
	}
	t.Logf("10/5 error:%v", rerr)
	rerr = sess.Call("/p/divide", &Arg{
		A: 10,
		B: 0,
	}, &result).Rerror()
	if rerr == nil {
		t.Fatal(rerr)
	}
	t.Logf("10/0 error:%v", rerr)
}
```

test command:

```sh
go test -v -run=TestBinder
```