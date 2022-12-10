# eRPC

[![tag](https://img.shields.io/github/tag/andeya/erpc.svg)](https://github.com/andeya/erpc/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.19-%23007d9c)
[![GoDoc](https://godoc.org/github.com/andeya/erpc?status.svg)](https://pkg.go.dev/github.com/andeya/erpc)
![Build Status](https://github.com/andeya/erpc/actions/workflows/go-ci.yml/badge.svg)
[![Go report](https://goreportcard.com/badge/github.com/andeya/erpc)](https://goreportcard.com/report/github.com/andeya/erpc)
[![License](https://img.shields.io/github/license/andeya/erpc)](./LICENSE)

eRPC is an efficient, extensible and easy-to-use RPC framework.

Suitable for RPC, Microservice, Peer-to-Peer, IM, Game and other fields.

[简体中文](https://github.com/andeya/erpc/tree/master/README_ZH.md)


![eRPC-Framework](https://github.com/andeya/erpc/raw/master/doc/erpc_module_diagram.png)


## Install


- go vesion ≥ 1.11

- install
```sh
GO111MODULE=on go get -u -v -insecure github.com/andeya/erpc/v7
```

- import
```go
import "github.com/andeya/erpc/v7"
```

## Feature

- Use peer to provide the same API package for the server and client
- Provide multi-layout abstractions such as:
  - peer
  - session/socket
  - router
  - handle/context
  - message
  - protocol
  - codec
  - transfer filter
  - plugin
- Support reboot and shutdown gracefully
- HTTP-compatible message format:
  - Composed of two parts, the `Header` and the `Body`
  - `Header` contains metadata in the same format as HTTP header
  - `Body` supports for custom codec of Content Type-Like, already implemented:
    - Protobuf
    - Thrift
    - JSON
    - XML
    - Form
    - Plain
  - Support push, call-reply and more message types
- Support custom message protocol, and provide some common implementations:
  - `rawproto` - Default high performance binary protocol
  - `jsonproto` - JSON message protocol
  - `pbproto` - Ptotobuf message protocol
  - `thriftproto` - Thrift message protocol
  - `httproto` - HTTP message protocol
- Optimized high performance transport layer
  - Use Non-block socket and I/O multiplexing technology
  - Support setting the size of socket I/O buffer
  - Support setting the size of the reading message (if exceed disconnect it)
  - Support controling the connection file descriptor
- Support a variety of network types:
  - `tcp`
  - `tcp4`
  - `tcp6`
  - `unix`
  - `unixpacket`
  - `kcp`
  - `quic`
  - other
    - websocket
    - evio
- Provide a rich plug-in point, and already implemented:
  - auth
  - binder
  - heartbeat
  - ignorecase(service method)
  - overloader
  - proxy(for unknown service method)
  - secure
- Powerful and flexible logging system:
  - Detailed log information, support print input and output details
  - Support setting slow operation alarm threshold
  - Support for custom implementation log component
- Client session support automatically redials after disconnection


## Benchmark

**Self Test**

- A server and a client process, running on the same machine
- CPU:    Intel Xeon E312xx (Sandy Bridge) 16 cores 2.53GHz
- Memory: 16G
- OS:     Linux 2.6.32-696.16.1.el6.centos.plus.x86_64, CentOS 6.4
- Go:     1.9.2
- Message size: 581 bytes
- Message codec: protobuf
- Sent total 1000000 messages

- erpc

| client concurrency | mean(ms) | median(ms) | max(ms) | min(ms) | throughput(TPS) |
| ------------------ | -------- | ---------- | ------- | ------- | --------------- |
| 100                | 1        | 0          | 16      | 0       | 75505           |
| 500                | 9        | 11         | 97      | 0       | 52192           |
| 1000               | 19       | 24         | 187     | 0       | 50040           |
| 2000               | 39       | 54         | 409     | 0       | 42551           |
| 5000               | 96       | 128        | 1148    | 0       | 46367           |

- erpc/socket

| client concurrency | mean(ms) | median(ms) | max(ms) | min(ms) | throughput(TPS) |
| ------------------ | -------- | ---------- | ------- | ------- | --------------- |
| 100                | 0        | 0          | 14      | 0       | 225682          |
| 500                | 2        | 1          | 24      | 0       | 212630          |
| 1000               | 4        | 3          | 51      | 0       | 180733          |
| 2000               | 8        | 6          | 64      | 0       | 183351          |
| 5000               | 21       | 18         | 651     | 0       | 133886          |

**Comparison Test**

<table>
<tr><th>Environment</th><th>Throughputs</th><th>Mean Latency</th><th>P99 Latency</th></tr>
<tr>
<td width="10%"><img src="https://github.com/andeya/rpc-benchmark/raw/master/result/env.png"></td>
<td width="30%"><img src="https://github.com/andeya/rpc-benchmark/raw/master/result/throughput.png"></td>
<td width="30%"><img src="https://github.com/andeya/rpc-benchmark/raw/master/result/mean_latency.png"></td>
<td width="30%"><img src="https://github.com/andeya/rpc-benchmark/raw/master/result/p99_latency.png"></td>
</tr>
</table>

**[More Detail](https://github.com/andeya/rpc-benchmark)**

- Profile torch of erpc/socket

![erpc_socket_profile_torch](https://github.com/andeya/erpc/raw/master/doc/erpc_socket_profile_torch.png)

**[svg file](https://github.com/andeya/erpc/raw/master/doc/erpc_socket_profile_torch.svg)**

- Heap torch of erpc/socket

![erpc_socket_heap_torch](https://github.com/andeya/erpc/raw/master/doc/erpc_socket_heap_torch.png)

**[svg file](https://github.com/andeya/erpc/raw/master/doc/erpc_socket_heap_torch.svg)**


## Example

### server.go

```go
package main

import (
	"fmt"
	"time"

	"github.com/andeya/erpc/v7"
)

func main() {
	defer erpc.FlushLogger()
	// graceful
	go erpc.GraceSignal()

	// server peer
	srv := erpc.NewPeer(erpc.PeerConfig{
		CountTime:   true,
		ListenPort:  9090,
		PrintDetail: true,
	})
	// srv.SetTLSConfig(erpc.GenerateTLSConfigForServer())

	// router
	srv.RouteCall(new(Math))

	// broadcast per 5s
	go func() {
		for {
			time.Sleep(time.Second * 5)
			srv.RangeSession(func(sess erpc.Session) bool {
				sess.Push(
					"/push/status",
					fmt.Sprintf("this is a broadcast, server time: %v", time.Now()),
				)
				return true
			})
		}
	}()

	// listen and serve
	srv.ListenAndServe()
}

// Math handler
type Math struct {
	erpc.CallCtx
}

// Add handles addition request
func (m *Math) Add(arg *[]int) (int, *erpc.Status) {
	// test meta
	erpc.Infof("author: %s", m.PeekMeta("author"))
	// add
	var r int
	for _, a := range *arg {
		r += a
	}
	// response
	return r, nil
}
```

### client.go

```go
package main

import (
  "time"

  "github.com/andeya/erpc/v7"
)

func main() {
  defer erpc.SetLoggerLevel("ERROR")()

  cli := erpc.NewPeer(erpc.PeerConfig{})
  defer cli.Close()
  // cli.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})

  cli.RoutePush(new(Push))

  sess, stat := cli.Dial(":9090")
  if !stat.OK() {
    erpc.Fatalf("%v", stat)
  }

  var result int
  stat = sess.Call("/math/add",
    []int{1, 2, 3, 4, 5},
    &result,
    erpc.WithAddMeta("author", "andeya"),
  ).Status()
  if !stat.OK() {
    erpc.Fatalf("%v", stat)
  }
  erpc.Printf("result: %d", result)
  erpc.Printf("Wait 10 seconds to receive the push...")
  time.Sleep(time.Second * 10)
}

// Push push handler
type Push struct {
  erpc.PushCtx
}

// Push handles '/push/status' message
func (p *Push) Status(arg *string) *erpc.Status {
  erpc.Printf("%s", *arg)
  return nil
}
```

[More Examples](https://github.com/andeya/erpc/tree/master/examples)

## Usage

**NOTE:**

- It is best to set the packet size when reading: `SetReadLimit`
- The default packet size limit when reading is 1 GB

### Peer(server or client) Demo

```go
// Start a server
var peer1 = erpc.NewPeer(erpc.PeerConfig{
ListenPort: 9090, // for server role
})
peer1.Listen()

...

// Start a client
var peer2 = erpc.NewPeer(erpc.PeerConfig{})
var sess, err = peer2.Dial("127.0.0.1:8080")
```

### Call-Struct API template

```go
type Aaa struct {
    erpc.CallCtx
}
func (x *Aaa) XxZz(arg *<T>) (<T>, *erpc.Status) {
    ...
    return r, nil
}
```

- register it to root router:

```go
// register the call route
// HTTP mapping: /aaa/xx_zz
// RPC mapping: Aaa.XxZz
peer.RouteCall(new(Aaa))

// or register the call route
// HTTP mapping: /xx_zz
// RPC mapping: XxZz
peer.RouteCallFunc((*Aaa).XxZz)
```

### Service method mapping

- The default mapping(HTTPServiceMethodMapper) of struct(func) name to service methods:
    - `AaBb` -> `/aa_bb`
    - `ABcXYz` -> `/abc_xyz`
    - `Aa__Bb` -> `/aa_bb`
    - `aa__bb` -> `/aa_bb`
    - `ABC__XYZ` -> `/abc_xyz`
    - `Aa_Bb` -> `/aa/bb`
    - `aa_bb` -> `/aa/bb`
    - `ABC_XYZ` -> `/abc/xyz`
    ```go
    erpc.SetServiceMethodMapper(erpc.HTTPServiceMethodMapper)
    ```

- The mapping(RPCServiceMethodMapper) of struct(func) name to service methods:
    - `AaBb` -> `AaBb`
    - `ABcXYz` -> `ABcXYz`
    - `Aa__Bb` -> `Aa_Bb`
    - `aa__bb` -> `aa_bb`
    - `ABC__XYZ` -> `ABC_XYZ`
    - `Aa_Bb` -> `Aa.Bb`
    - `aa_bb` -> `aa.bb`
    - `ABC_XYZ` -> `ABC.XYZ`
    ```go
    erpc.SetServiceMethodMapper(erpc.RPCServiceMethodMapper)
    ```

### Call-Function API template

```go
func XxZz(ctx erpc.CallCtx, arg *<T>) (<T>, *erpc.Status) {
    ...
    return r, nil
}
```

- register it to root router:

```go
// register the call route
// HTTP mapping: /xx_zz
// RPC mapping: XxZz
peer.RouteCallFunc(XxZz)
```

### Push-Struct API template

```go
type Bbb struct {
    erpc.PushCtx
}
func (b *Bbb) YyZz(arg *<T>) *erpc.Status {
    ...
    return nil
}
```

- register it to root router:

```go
// register the push handler
// HTTP mapping: /bbb/yy_zz
// RPC mapping: Bbb.YyZz
peer.RoutePush(new(Bbb))

// or register the push handler
// HTTP mapping: /yy_zz
// RPC mapping: YyZz
peer.RoutePushFunc((*Bbb).YyZz)
```

### Push-Function API template

```go
// YyZz register the handler
func YyZz(ctx erpc.PushCtx, arg *<T>) *erpc.Status {
    ...
    return nil
}
```

- register it to root router:

```go
// register the push handler
// HTTP mapping: /yy_zz
// RPC mapping: YyZz
peer.RoutePushFunc(YyZz)
```

### Unknown-Call-Function API template

```go
func XxxUnknownCall (ctx erpc.UnknownCallCtx) (interface{}, *erpc.Status) {
    ...
    return r, nil
}
```

- register it to root router:

```go
// register the unknown call route: /*
peer.SetUnknownCall(XxxUnknownCall)
```

### Unknown-Push-Function API template

```go
func XxxUnknownPush(ctx erpc.UnknownPushCtx) *erpc.Status {
    ...
    return nil
}
```

- register it to root router:

```go
// register the unknown push route: /*
peer.SetUnknownPush(XxxUnknownPush)
```

### Plugin Demo

```go
// NewIgnoreCase Returns a ignoreCase plugin.
func NewIgnoreCase() *ignoreCase {
    return &ignoreCase{}
}

type ignoreCase struct{}

var (
    _ erpc.PostReadCallHeaderPlugin = new(ignoreCase)
    _ erpc.PostReadPushHeaderPlugin = new(ignoreCase)
)

func (i *ignoreCase) Name() string {
    return "ignoreCase"
}

func (i *ignoreCase) PostReadCallHeader(ctx erpc.ReadCtx) *erpc.Status {
    // Dynamic transformation path is lowercase
    ctx.UriObject().Path = strings.ToLower(ctx.UriObject().Path)
    return nil
}

func (i *ignoreCase) PostReadPushHeader(ctx erpc.ReadCtx) *erpc.Status {
    // Dynamic transformation path is lowercase
    ctx.UriObject().Path = strings.ToLower(ctx.UriObject().Path)
    return nil
}
```

### Register above handler and plugin

```go
// add router group
group := peer.SubRoute("test")
// register to test group
group.RouteCall(new(Aaa), NewIgnoreCase())
peer.RouteCallFunc(XxZz, NewIgnoreCase())
group.RoutePush(new(Bbb))
peer.RoutePushFunc(YyZz)
peer.SetUnknownCall(XxxUnknownCall)
peer.SetUnknownPush(XxxUnknownPush)
```

### Config

```go
type PeerConfig struct {
    Network            string        `yaml:"network"              ini:"network"              comment:"Network; tcp, tcp4, tcp6, unix, unixpacket, kcp or quic"`
    LocalIP            string        `yaml:"local_ip"             ini:"local_ip"             comment:"Local IP"`
    ListenPort         uint16        `yaml:"listen_port"          ini:"listen_port"          comment:"Listen port; for server role"`
    DialTimeout time.Duration `yaml:"dial_timeout" ini:"dial_timeout" comment:"Default maximum duration for dialing; for client role; ns,µs,ms,s,m,h"`
    RedialTimes        int32         `yaml:"redial_times"         ini:"redial_times"         comment:"The maximum times of attempts to redial, after the connection has been unexpectedly broken; Unlimited when <0; for client role"`
	RedialInterval     time.Duration `yaml:"redial_interval"      ini:"redial_interval"      comment:"Interval of redialing each time, default 100ms; for client role; ns,µs,ms,s,m,h"`
    DefaultBodyCodec   string        `yaml:"default_body_codec"   ini:"default_body_codec"   comment:"Default body codec type id"`
    DefaultSessionAge  time.Duration `yaml:"default_session_age"  ini:"default_session_age"  comment:"Default session max age, if less than or equal to 0, no time limit; ns,µs,ms,s,m,h"`
    DefaultContextAge  time.Duration `yaml:"default_context_age"  ini:"default_context_age"  comment:"Default CALL or PUSH context max age, if less than or equal to 0, no time limit; ns,µs,ms,s,m,h"`
    SlowCometDuration  time.Duration `yaml:"slow_comet_duration"  ini:"slow_comet_duration"  comment:"Slow operation alarm threshold; ns,µs,ms,s ..."`
    PrintDetail        bool          `yaml:"print_detail"         ini:"print_detail"         comment:"Is print body and metadata or not"`
    CountTime          bool          `yaml:"count_time"           ini:"count_time"           comment:"Is count cost time or not"`
}
```

### Optimize

- SetMessageSizeLimit sets max message size.
  If maxSize<=0, set it to max uint32.

    ```go
    func SetMessageSizeLimit(maxMessageSize uint32)
    ```

- SetSocketKeepAlive sets whether the operating system should send
  keepalive messages on the connection.

    ```go
    func SetSocketKeepAlive(keepalive bool)
    ```

- SetSocketKeepAlivePeriod sets period between keep alives.

    ```go
    func SetSocketKeepAlivePeriod(d time.Duration)
    ```

- SetSocketNoDelay controls whether the operating system should delay
  message transmission in hopes of sending fewer messages (Nagle's
  algorithm).  The default is true (no delay), meaning that data is
  sent as soon as possible after a Write.

    ```go
    func SetSocketNoDelay(_noDelay bool)
    ```

- SetSocketReadBuffer sets the size of the operating system's
  receive buffer associated with the connection.

    ```go
    func SetSocketReadBuffer(bytes int)
    ```

- SetSocketWriteBuffer sets the size of the operating system's
  transmit buffer associated with the connection.

    ```go
    func SetSocketWriteBuffer(bytes int)
    ```


## Extensions

### Codec

| package                                  | import                                   | description                  |
| ---------------------------------------- | ---------------------------------------- | ---------------------------- |
| [json](https://github.com/andeya/erpc/blob/master/codec/json_codec.go) | `"github.com/andeya/erpc/v7/codec"` | JSON codec(erpc own)     |
| [protobuf](https://github.com/andeya/erpc/blob/master/codec/protobuf_codec.go) | `"github.com/andeya/erpc/v7/codec"` | Protobuf codec(erpc own) |
| [thrift](https://github.com/andeya/erpc/blob/master/codec/thrift_codec.go) | `"github.com/andeya/erpc/v7/codec"` | Form(url encode) codec(erpc own)   |
| [xml](https://github.com/andeya/erpc/blob/master/codec/xml_codec.go) | `"github.com/andeya/erpc/v7/codec"` | Form(url encode) codec(erpc own)   |
| [plain](https://github.com/andeya/erpc/blob/master/codec/plain_codec.go) | `"github.com/andeya/erpc/v7/codec"` | Plain text codec(erpc own)   |
| [form](https://github.com/andeya/erpc/blob/master/codec/form_codec.go) | `"github.com/andeya/erpc/v7/codec"` | Form(url encode) codec(erpc own)   |

### Plugin

| package                                  | import                                   | description                              |
| ---------------------------------------- | ---------------------------------------- | ---------------------------------------- |
| [auth](https://github.com/andeya/erpc/tree/master/plugin/auth) | `"github.com/andeya/erpc/v7/plugin/auth"` | An auth plugin for verifying peer at the first time |
| [binder](https://github.com/andeya/erpc/tree/master/plugin/binder) | `"github.com/andeya/erpc/v7/plugin/binder"` | Parameter Binding Verification for Struct Handler |
| [heartbeat](https://github.com/andeya/erpc/tree/master/plugin/heartbeat) | `"github.com/andeya/erpc/v7/plugin/heartbeat"` | A generic timing heartbeat plugin        |
| [proxy](https://github.com/andeya/erpc/tree/master/plugin/proxy) | `"github.com/andeya/erpc/v7/plugin/proxy"` | A proxy plugin for handling unknown calling or pushing |
[secure](https://github.com/andeya/erpc/tree/master/plugin/secure)|`"github.com/andeya/erpc/v7/plugin/secure"` | Encrypting/decrypting the message body
[overloader](https://github.com/andeya/erpc/tree/master/plugin/overloader)|`"github.com/andeya/erpc/v7/plugin/overloader"` | A plugin to protect erpc from overload

### Protocol

| package                                  | import                                   | description                              |
| ---------------------------------------- | ---------------------------------------- | ---------------------------------------- |
| [rawproto](https://github.com/andeya/erpc/tree/master/proto/rawproto) | `"github.com/andeya/erpc/v7/proto/rawproto` | A fast socket communication protocol(erpc default protocol) |
| [jsonproto](https://github.com/andeya/erpc/tree/master/proto/jsonproto) | `"github.com/andeya/erpc/v7/proto/jsonproto"` | A JSON socket communication protocol     |
| [pbproto](https://github.com/andeya/erpc/tree/master/proto/pbproto) | `"github.com/andeya/erpc/v7/proto/pbproto"` | A Protobuf socket communication protocol     |
| [thriftproto](https://github.com/andeya/erpc/tree/master/proto/thriftproto) | `"github.com/andeya/erpc/v7/proto/thriftproto"` | A Thrift communication protocol     |
| [httproto](https://github.com/andeya/erpc/tree/master/proto/httproto) | `"github.com/andeya/erpc/v7/proto/httproto"` | A HTTP style socket communication protocol     |

### Transfer-Filter

| package                                  | import                                   | description                              |
| ---------------------------------------- | ---------------------------------------- | ---------------------------------------- |
| [gzip](https://github.com/andeya/erpc/tree/master/xfer/gzip) | `"github.com/andeya/erpc/v7/xfer/gzip"` | Gzip(erpc own)                       |
| [md5](https://github.com/andeya/erpc/tree/master/xfer/md5) | `"github.com/andeya/erpc/v7/xfer/md5"` | Provides a integrity check transfer filter |

### Mixer

| package                                  | import                                   | description                              |
| ---------------------------------------- | ---------------------------------------- | ---------------------------------------- |
| [multiclient](https://github.com/andeya/erpc/tree/master/mixer/multiclient) | `"github.com/andeya/erpc/v7/mixer/multiclient"` | Higher throughput client connection pool when transferring large messages (such as downloading files) |
| [websocket](https://github.com/andeya/erpc/tree/master/mixer/websocket) | `"github.com/andeya/erpc/v7/mixer/websocket"` | Makes the eRPC framework compatible with websocket protocol as specified in RFC 6455 |
| [evio](https://github.com/andeya/erpc/tree/master/mixer/evio) | `"github.com/andeya/erpc/v7/mixer/evio"` | A fast event-loop networking framework that uses the erpc API layer |
| [html](https://github.com/xiaoenai/tp-micro/tree/master/helper/mod-html) | `html "github.com/xiaoenai/tp-micro/helper/mod-html"` | HTML render for http client |

## Projects based on eRPC

| project                                  | description                              |
| ---------------------------------------- | ---------------------------------------- |
| [TP-Micro](https://github.com/xiaoenai/tp-micro) | TP-Micro is a simple, powerful micro service framework based on eRPC |
| [Pholcus](https://github.com/andeya/pholcus) | Pholcus is a distributed, high concurrency and powerful web crawler software |

## Business Users

<a href="http://www.xiaoenai.com"><img src="https://raw.githubusercontent.com/andeya/imgs-repo/master/xiaoenai.png" height="50" alt="深圳市梦之舵信息技术有限公司"/></a>
&nbsp;&nbsp;
<a href="https://tech.pingan.com/index.html"><img src="http://pa-tech.hirede.com/templates/pa-tech/Images/logo.png" height="50" alt="平安科技"/></a>
<br/>
<a href="http://www.fun.tv"><img src="http://static.funshion.com/open/static/img/logo.gif" height="70" alt="北京风行在线技术有限公司"/></a>
&nbsp;&nbsp;
<a href="http://www.kejishidai.cn"><img src="http://simg.ktvms.com/picture/logo.png" height="70" alt="北京可即时代网络公司"/></a>
<a href="https://www.kuaishou.com/"><img src="https://inews.gtimg.com/newsapp_bt/0/4400789257/1000" height="70" alt="快手短视频平台"/></a>

## License

eRPC is under Apache v2 License. See the [LICENSE](https://github.com/andeya/erpc/raw/master/LICENSE) file for the full license text
