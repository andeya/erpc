## proxy

A plugin for handling unknown calling or pushing.

#### Demo

```go
package main

import (
	"time"

	"github.com/henrylee2cn/tp/v6"
	"github.com/henrylee2cn/tp/v6/plugin/proxy"
)

func main() {
	defer tp.FlushLogger()
	srv := tp.NewPeer(
		tp.PeerConfig{
			ListenPort: 8080,
		},
		newUnknownProxy(),
	)
	srv.ListenAndServe()
}

func newUnknownProxy() tp.Plugin {
	cli := tp.NewPeer(tp.PeerConfig{RedialTimes: 3})
	var sess tp.Session
	var stat *tp.Status
DIAL:
	sess, stat = cli.Dial(":9090")
	if !stat.OK() {
		tp.Warnf("%v", stat)
		time.Sleep(time.Second * 3)
		goto DIAL
	}
	return proxy.NewPlugin(func(*proxy.Label) proxy.Forwarder {
		return sess
	})
}
```
