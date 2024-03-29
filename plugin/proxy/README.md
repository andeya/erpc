## proxy

A plugin for handling unknown calling or pushing.

#### Demo

```go
package main

import (
	"time"

	"github.com/andeya/erpc/v7"
	"github.com/andeya/erpc/v7/plugin/proxy"
)

func main() {
	defer erpc.FlushLogger()
	srv := erpc.NewPeer(
		erpc.PeerConfig{
			ListenPort: 8080,
		},
		newUnknownProxy(),
	)
	srv.ListenAndServe()
}

func newUnknownProxy() erpc.Plugin {
	cli := erpc.NewPeer(erpc.PeerConfig{RedialTimes: 3})
	var sess erpc.Session
	var stat *erpc.Status
DIAL:
	sess, stat = cli.Dial(":9090")
	if !stat.OK() {
		erpc.Warnf("%v", stat)
		time.Sleep(time.Second * 3)
		goto DIAL
	}
	return proxy.NewPlugin(func(*proxy.Label) proxy.Forwarder {
		return sess
	})
}
```
