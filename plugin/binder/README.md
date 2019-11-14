## binder

Parameter Binding Verification Plugin for Struct Handler.

### Usage

`import "github.com/henrylee2cn/erpc/v6/plugin/binder"`

#### Param-Tags

tag   |   key    | required |     value     |   desc
------|----------|----------|---------------|----------------------------------
param |   meta    | no |  (name e.g.`param:"<meta:id>"`)  | It indicates that the parameter is from the meta.
param |   swap    | no |   name (e.g.`param:"<swap:id>"`)  | It indicates that the parameter is from the context swap.
param |   desc   |      no      |     (e.g.`param:"<desc:id>"`)   | Parameter Description
param |   len    |      no      |   (e.g.`param:"<len:3:6>"`)  | Length range [a,b] of parameter's value
param |   range  |      no      |   (e.g.`param:"<range:0:10>"`)   | Numerical range [a,b] of parameter's value
param |  nonzero |      no      |    -    | Not allowed to zero
param |  regexp  |      no      |   (e.g.`param:"<regexp:^\\w+$>"`)  | Regular expression validation
param |   stat   |      no      |(e.g.`param:"<stat:100002:wrong password format>"`)| Custom error code and message

NOTES:

* `param:"-"` means ignore
* Encountered untagged exportable anonymous structure field, automatic recursive resolution
* Parameter name is the name of the structure field converted to snake format
* If the parameter is not from `meta` or `swap`, it is the default from the body
* Support for multiple rule combinations, e.g.`param:"<regexp:^\\w+$><len:6:8><stat:100002:wrong password format>"`

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

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/plugin/binder"
)

type (
	Arg struct {
		A int
		B int    `param:"<range:1:100>"`
		C string `param:"<regexp:^[1-9]\\d*$>"`
		Query
		XyZ       string  `param:"<meta><nonzero><stat: 100002: Parameter cannot be empty>"`
		SwapValue float32 `param:"<swap><nonzero>"`
	}
	Query struct {
		X string `param:"<meta:_x>"`
	}
)

type P struct{ erpc.CallCtx }

func (p *P) Divide(arg *Arg) (int, *erpc.Status) {
	erpc.Infof("meta arg _x: %s, xy_z: %s, swap_value: %v", arg.Query.X, arg.XyZ, arg.SwapValue)
	return arg.A / arg.B, nil
}

type SwapPlugin struct{}

func (s *SwapPlugin) Name() string {
	return "swap_plugin"
}
func (s *SwapPlugin) PostReadCallBody(ctx erpc.ReadCtx) *erpc.Status {
	ctx.Swap().Store("swap_value", 123)
	return nil
}

func TestBinder(t *testing.T) {
	bplugin := binder.NewStructArgsBinder(nil)
	srv := erpc.NewPeer(
		erpc.PeerConfig{ListenPort: 9090},
	)
	srv.PluginContainer().AppendRight(bplugin)
	srv.RouteCall(new(P), new(SwapPlugin))
	go srv.ListenAndServe()
	time.Sleep(time.Second)

	cli := erpc.NewPeer(erpc.PeerConfig{})
	sess, stat := cli.Dial(":9090")
	if !stat.OK() {
		t.Fatal(stat)
	}
	var result int
	stat = sess.Call("/p/divide", &Arg{
		A: 10,
		B: 2,
		C: "3",
	},
		&result,
		erpc.WithSetMeta("_x", "testmeta_x"),
		erpc.WithSetMeta("xy_z", "testmeta_xy_z"),
	).Status()
	if !stat.OK() {
		t.Fatal(stat)
	}
	t.Logf("10/2=%d", result)
	stat = sess.Call("/p/divide", &Arg{
		A: 10,
		B: 5,
		C: "3",
	}, &result).Status()
	if stat.OK() {
		t.Fatal(stat)
	}
	t.Logf("10/5 error:%v", stat)
	stat = sess.Call("/p/divide", &Arg{
		A: 10,
		B: 0,
		C: "3",
	}, &result).Status()
	if stat.OK() {
		t.Fatal(stat)
	}
	t.Logf("10/0 error:%v", stat)
}
```

test command:

```sh
go test -v -run=TestBinder
```