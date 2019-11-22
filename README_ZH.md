# eRPC [![GitHub release](https://img.shields.io/github/release/henrylee2cn/erpc.svg?style=flat-square)](https://github.com/henrylee2cn/erpc/releases) [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/erpc?style=flat-square)](http://goreportcard.com/report/henrylee2cn/erpc) [![github issues](https://img.shields.io/github/issues/henrylee2cn/erpc.svg?style=flat-square)](https://github.com/henrylee2cn/erpc/issues?q=is%3Aopen+is%3Aissue) [![github closed issues](https://img.shields.io/github/issues-closed-raw/henrylee2cn/erpc.svg?style=flat-square)](https://github.com/henrylee2cn/erpc/issues?q=is%3Aissue+is%3Aclosed) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/henrylee2cn/erpc) [![view examples](https://img.shields.io/badge/learn%20by-examples-00BCD4.svg?style=flat-square)](https://github.com/henrylee2cn/erpc/tree/master/examples)
<!-- [![view Go网络编程群](https://img.shields.io/badge/官方QQ群-Go网络编程(42730308)-27a5ea.svg?style=flat-square)](http://jq.qq.com/?_wv=1027&k=fzi4p1) -->


eRPC 是一个高效、可扩展且简单易用的 RPC 框架。

适用于 RPC、微服务、点对点长连接、IM 和游戏等领域。


![eRPC-Framework](https://github.com/henrylee2cn/erpc/raw/master/doc/erpc_module_diagram.png)


## 安装

- environment GO111MODULE=on
- go vesion ≥ 1.11

```go
import "github.com/henrylee2cn/erpc/v6"
```

## 特性

- 使用 peer 为 server 和 client 提供相同的 API 封装
- 提供多层抽象，如：
  - peer
  - session/socket
  - router
  - handle/context
  - message
  - protocol
  - codec
  - transfer filter
  - plugin
- 支持平滑重启和关闭
- 兼容 HTTP 的消息格式：
  - 由 `Header` 和 `Body` 两部分组成
  - `Header` 包含与 HTTP header 格式相同的 metadata
  - `Body` 支持类似 Content Type 的自定义编解码器，已经实现的：
    - Protobuf
    - Thrift
    - JSON
    - XML
    - Form
    - Plain
  - 支持 push、call-reply 和更多的消息类型
- 支持自定义消息协议，并提供了一些常见实现：
  - `rawproto` - 默认的高性能二进制协议
  - `jsonproto` - JSON 消息协议
  - `pbproto` - Ptotobuf 消息协议
  - `thriftproto` - Thrift 消息协议
  - `httproto` - HTTP 消息协议
- 可优化的高性能传输层
  - 使用 Non-block socket 和 I/O 多路复用技术
  - 支持设置套接字 I/O 的缓冲区大小
  - 支持设置读取消息的大小（如果超过则断开连接）
  - 支持控制连接的文件描述符
- 支持多种网络类型：
  - `tcp`
  - `tcp4`
  - `tcp6`
  - `unix`
  - `unixpacket`
  - `quic`
  - 其他
    - websocket
    - evio
- 提供丰富的插件埋点，并已实现：
  - auth
  - binder
  - heartbeat
  - ignorecase(service method)
  - overloader
  - proxy(for unknown service method)
  - secure
- 强大灵活的日志系统：
  - 详细的日志信息，支持打印输入和输出详细信息
  - 支持设置慢操作警报阈值
  - 支持自定义实现日志组件
- 客户端会话支持在断开连接后自动重拨


## 性能测试

**自测**

- 一个服务端与一个客户端进程，在同一台机器上运行
- CPU:    Intel Xeon E312xx (Sandy Bridge) 16 cores 2.53GHz
- Memory: 16G
- OS:     Linux 2.6.32-696.16.1.el6.centos.plus.x86_64, CentOS 6.4
- Go:     1.9.2
- 信息大小: 581 bytes
- 信息编码：protobuf
- 发送 1000000 条信息

- erpc

| 并发client | 平均值(ms) | 中位数(ms) | 最大值(ms) | 最小值(ms) | 吞吐率(TPS) |
| -------- | ------- | ------- | ------- | ------- | -------- |
| 100      | 1       | 0       | 16      | 0       | 75505    |
| 500      | 9       | 11      | 97      | 0       | 52192    |
| 1000     | 19      | 24      | 187     | 0       | 50040    |
| 2000     | 39      | 54      | 409     | 0       | 42551    |
| 5000     | 96      | 128     | 1148    | 0       | 46367    |

- erpc/socket

| 并发client | 平均值(ms) | 中位数(ms) | 最大值(ms) | 最小值(ms) | 吞吐率(TPS) |
| -------- | ------- | ------- | ------- | ------- | -------- |
| 100      | 0       | 0       | 14      | 0       | 225682   |
| 500      | 2       | 1       | 24      | 0       | 212630   |
| 1000     | 4       | 3       | 51      | 0       | 180733   |
| 2000     | 8       | 6       | 64      | 0       | 183351   |
| 5000     | 21      | 18      | 651     | 0       | 133886   |

**对比测试**

<table>
<tr><th>Environment</th><th>Throughputs</th><th>Mean Latency</th><th>P99 Latency</th></tr>
<tr>
<td width="10%"><img src="https://github.com/henrylee2cn/rpc-benchmark/raw/master/result/env.png"></td>
<td width="30%"><img src="https://github.com/henrylee2cn/rpc-benchmark/raw/master/result/throughput.png"></td>
<td width="30%"><img src="https://github.com/henrylee2cn/rpc-benchmark/raw/master/result/mean_latency.png"></td>
<td width="30%"><img src="https://github.com/henrylee2cn/rpc-benchmark/raw/master/result/p99_latency.png"></td>
</tr>
</table>

**[More Detail](https://github.com/henrylee2cn/rpc-benchmark)**

- CPU耗时火焰图 erpc/socket

![erpc_socket_profile_torch](https://github.com/henrylee2cn/erpc/raw/master/doc/erpc_socket_profile_torch.png)

**[svg file](https://github.com/henrylee2cn/erpc/raw/master/doc/erpc_socket_profile_torch.svg)**

- 堆栈信息火焰图 erpc/socket

![erpc_socket_heap_torch](https://github.com/henrylee2cn/erpc/raw/master/doc/erpc_socket_heap_torch.png)

**[svg file](https://github.com/henrylee2cn/erpc/raw/master/doc/erpc_socket_heap_torch.svg)**


## 代码示例

### server.go

```go
package main

import (
	"fmt"
	"time"

	"github.com/henrylee2cn/erpc/v6"
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

	"github.com/henrylee2cn/erpc/v6"
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
		erpc.WithAddMeta("author", "henrylee2cn"),
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

[更多示例](https://github.com/henrylee2cn/erpc/blob/master/examples)


## 用法

### Peer端点（服务端或客户端）示例

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

### 自带ServiceMethod映射规则

- 结构体或方法名称到服务方法名称的默认映射（HTTPServiceMethodMapper）：
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

- 结构体或方法名称到服务方法名称的映射（RPCServiceMethodMapper）：
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

### Call-Struct 接口模版

```go
type Aaa struct {
    erpc.CallCtx
}
func (x *Aaa) XxZz(arg *<T>) (<T>, *erpc.Status) {
    ...
    return r, nil
}
```

- 注册到根路由：

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

### Call-Function 接口模板

```go
func XxZz(ctx erpc.CallCtx, arg *<T>) (<T>, *erpc.Status) {
    ...
    return r, nil
}
```

- 注册到根路由：

```go
// register the call route
// HTTP mapping: /xx_zz
// RPC mapping: XxZz
peer.RouteCallFunc(XxZz)
```

### Push-Struct 接口模板

```go
type Bbb struct {
    erpc.PushCtx
}
func (b *Bbb) YyZz(arg *<T>) *erpc.Status {
    ...
    return nil
}
```

- 注册到根路由：

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

### Push-Function 接口模板

```go
// YyZz register the handler
func YyZz(ctx erpc.PushCtx, arg *<T>) *erpc.Status {
    ...
    return nil
}
```

- 注册到根路由：

```go
// register the push handler
// HTTP mapping: /yy_zz
// RPC mapping: YyZz
peer.RoutePushFunc(YyZz)
```

### Unknown-Call-Function 接口模板

```go
func XxxUnknownCall (ctx erpc.UnknownCallCtx) (interface{}, *erpc.Status) {
    ...
    return r, nil
}
```

- 注册到根路由：

```go
// register the unknown pull route: /*
peer.SetUnknownCall(XxxUnknownCall)
```

### Unknown-Push-Function 接口模板

```go
func XxxUnknownPush(ctx erpc.UnknownPushCtx) *erpc.Status {
    ...
    return nil
}
```

- 注册到根路由：

```go
// register the unknown push route: /*
peer.SetUnknownPush(XxxUnknownPush)
```

### 插件示例

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

### 注册以上操作和插件示例到路由

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

### 配置信息

```go
type PeerConfig struct {
    Network            string        `yaml:"network"              ini:"network"              comment:"Network; tcp, tcp4, tcp6, unix, unixpacket or quic"`
    LocalIP            string        `yaml:"local_ip"             ini:"local_ip"             comment:"Local IP"`
    ListenPort         uint16        `yaml:"listen_port"          ini:"listen_port"          comment:"Listen port; for server role"`
    DialTimeout time.Duration `yaml:"dial_timeout" ini:"dial_timeout" comment:"Default maximum duration for dialing; for client role; ns,µs,ms,s,m,h"`
    RedialTimes        int32         `yaml:"redial_times"         ini:"redial_times"         comment:"The maximum times of attempts to redial, after the connection has been unexpectedly broken; Unlimited when <0; for client role"`
	RedialInterval     time.Duration `yaml:"redial_interval"      ini:"redial_interval"      comment:"Interval of redialing each time, default 100ms; for client role; ns,µs,ms,s,m,h"`
    DefaultBodyCodec   string        `yaml:"default_body_codec"   ini:"default_body_codec"   comment:"Default body codec type id"`
    DefaultSessionAge  time.Duration `yaml:"default_session_age"  ini:"default_session_age"  comment:"Default session max age, if less than or equal to 0, no time limit; ns,µs,ms,s,m,h"`
    DefaultContextAge  time.Duration `yaml:"default_context_age"  ini:"default_context_age"  comment:"Default PULL or PUSH context max age, if less than or equal to 0, no time limit; ns,µs,ms,s,m,h"`
    SlowCometDuration  time.Duration `yaml:"slow_comet_duration"  ini:"slow_comet_duration"  comment:"Slow operation alarm threshold; ns,µs,ms,s ..."`
    PrintDetail        bool          `yaml:"print_detail"         ini:"print_detail"         comment:"Is print body and metadata or not"`
    CountTime          bool          `yaml:"count_time"           ini:"count_time"           comment:"Is count cost time or not"`
}
```

### 通信优化

- SetMessageSizeLimit 设置报文大小的上限，
  如果 maxSize<=0，上限默认为最大 uint32

    ```go
    func SetMessageSizeLimit(maxMessageSize uint32)
    ```

- SetSocketKeepAlive 是否允许操作系统的发送TCP的keepalive探测包

    ```go
    func SetSocketKeepAlive(keepalive bool)
    ```


- SetSocketKeepAlivePeriod 设置操作系统的TCP发送keepalive探测包的频度

    ```go
    func SetSocketKeepAlivePeriod(d time.Duration)
    ```

- SetSocketNoDelay 是否禁用Nagle算法，禁用后将不在合并较小数据包进行批量发送，默认为禁用

    ```go
    func SetSocketNoDelay(_noDelay bool)
    ```

- SetSocketReadBuffer 设置操作系统的TCP读缓存区的大小

    ```go
    func SetSocketReadBuffer(bytes int)
    ```

- SetSocketWriteBuffer 设置操作系统的TCP写缓存区的大小

    ```go
    func SetSocketWriteBuffer(bytes int)
    ```


## 扩展包

### 编解码器
| package                                  | import                                   | description                  |
| ---------------------------------------- | ---------------------------------------- | ---------------------------- |
| [json](https://github.com/henrylee2cn/erpc/blob/master/codec/json_codec.go) | `"github.com/henrylee2cn/erpc/v6/codec"` | JSON codec(erpc own)     |
| [protobuf](https://github.com/henrylee2cn/erpc/blob/master/codec/protobuf_codec.go) | `"github.com/henrylee2cn/erpc/v6/codec"` | Protobuf codec(erpc own) |
| [thrift](https://github.com/henrylee2cn/erpc/blob/master/codec/thrift_codec.go) | `"github.com/henrylee2cn/erpc/v6/codec"` | Form(url encode) codec(erpc own)   |
| [xml](https://github.com/henrylee2cn/erpc/blob/master/codec/xml_codec.go) | `"github.com/henrylee2cn/erpc/v6/codec"` | Form(url encode) codec(erpc own)   |
| [plain](https://github.com/henrylee2cn/erpc/blob/master/codec/plain_codec.go) | `"github.com/henrylee2cn/erpc/v6/codec"` | Plain text codec(erpc own)   |
| [form](https://github.com/henrylee2cn/erpc/blob/master/codec/form_codec.go) | `"github.com/henrylee2cn/erpc/v6/codec"` | Form(url encode) codec(erpc own)   |

### 插件

| package                                  | import                                   | description                              |
| ---------------------------------------- | ---------------------------------------- | ---------------------------------------- |
| [auth](https://github.com/henrylee2cn/erpc/tree/master/plugin/auth) | `"github.com/henrylee2cn/erpc/v6/plugin/auth"` | An auth plugin for verifying peer at the first time |
| [binder](https://github.com/henrylee2cn/erpc/tree/master/plugin/binder) | `"github.com/henrylee2cn/erpc/v6/plugin/binder"` | Parameter Binding Verification for Struct Handler |
| [heartbeat](https://github.com/henrylee2cn/erpc/tree/master/plugin/heartbeat) | `"github.com/henrylee2cn/erpc/v6/plugin/heartbeat"` | A generic timing heartbeat plugin        |
| [proxy](https://github.com/henrylee2cn/erpc/tree/master/plugin/proxy) | `"github.com/henrylee2cn/erpc/v6/plugin/proxy"` | A proxy plugin for handling unknown calling or pushing |
[secure](https://github.com/henrylee2cn/erpc/tree/master/plugin/secure)|`"github.com/henrylee2cn/erpc/v6/plugin/secure"` | Encrypting/decrypting the message body
[overloader](https://github.com/henrylee2cn/erpc/tree/master/plugin/overloader)|`"github.com/henrylee2cn/erpc/v6/plugin/overloader"` | A plugin to protect erpc from overload

### 协议

| package                                  | import                                   | description                              |
| ---------------------------------------- | ---------------------------------------- | ---------------------------------------- |
| [rawproto](https://github.com/henrylee2cn/erpc/tree/master/proto/rawproto) | `"github.com/henrylee2cn/erpc/v6/proto/rawproto` | 一个高性能的通信协议（erpc默认）|
| [jsonproto](https://github.com/henrylee2cn/erpc/tree/master/proto/jsonproto) | `"github.com/henrylee2cn/erpc/v6/proto/jsonproto"` | JSON 格式的通信协议     |
| [pbproto](https://github.com/henrylee2cn/erpc/tree/master/proto/pbproto) | `"github.com/henrylee2cn/erpc/v6/proto/pbproto"` | Protobuf 格式的通信协议     |
| [thriftproto](https://github.com/henrylee2cn/erpc/tree/master/proto/thriftproto) | `"github.com/henrylee2cn/erpc/v6/proto/thriftproto"` | Thrift 格式的通信协议     |
| [httproto](https://github.com/henrylee2cn/erpc/tree/master/proto/httproto) | `"github.com/henrylee2cn/erpc/v6/proto/httproto"` | HTTP 格式的通信协议     |

### 传输过滤器

| package                                  | import                                   | description                              |
| ---------------------------------------- | ---------------------------------------- | ---------------------------------------- |
| [gzip](https://github.com/henrylee2cn/erpc/tree/master/xfer/gzip) | `"github.com/henrylee2cn/erpc/v6/xfer/gzip"` | Gzip(erpc own)                       |
| [md5](https://github.com/henrylee2cn/erpc/tree/master/xfer/md5) | `"github.com/henrylee2cn/erpc/v6/xfer/md5"` | Provides a integrity check transfer filter |

### 其他模块

| package                                  | import                                   | description                              |
| ---------------------------------------- | ---------------------------------------- | ---------------------------------------- |
| [multiclient](https://github.com/henrylee2cn/erpc/tree/master/mixer/multiclient) | `"github.com/henrylee2cn/erpc/v6/mixer/multiclient"` | Higher throughput client connection pool when transferring large messages (such as downloading files) |
| [websocket](https://github.com/henrylee2cn/erpc/tree/master/mixer/websocket) | `"github.com/henrylee2cn/erpc/v6/mixer/websocket"` | Makes the eRPC framework compatible with websocket protocol as specified in RFC 6455 |
| [evio](https://github.com/henrylee2cn/erpc/tree/master/mixer/evio) | `"github.com/henrylee2cn/erpc/v6/mixer/evio"` | A fast event-loop networking framework that uses the erpc API layer |
| [html](https://github.com/xiaoenai/tp-micro/tree/master/helper/mod-html) | `html "github.com/xiaoenai/tp-micro/helper/mod-html"` | HTML render for http client |

## 基于eRPC的项目

| project                                  | description                              |
| ---------------------------------------- | ---------------------------------------- |
| [TP-Micro](https://github.com/xiaoenai/tp-micro) | TP-Micro 是一个基于 eRPC 定制的、简约而强大的微服务框架          |
| [Pholcus](https://github.com/henrylee2cn/pholcus) | Pholcus（幽灵蛛）是一款纯Go语言编写的支持分布式的高并发、重量级爬虫软件，定位于互联网数据采集，为具备一定Go或JS编程基础的人提供一个只需关注规则定制的功能强大的爬虫工具 |

## 企业用户

<a href="http://www.xiaoenai.com"><img src="https://raw.githubusercontent.com/henrylee2cn/imgs-repo/master/xiaoenai.png" height="50" alt="深圳市梦之舵信息技术有限公司"/></a>
&nbsp;&nbsp;
<a href="https://tech.pingan.com/index.html"><img src="http://pa-tech.hirede.com/templates/pa-tech/Images/logo.png" height="50" alt="平安科技"/></a>
<br/>
<a href="http://www.fun.tv"><img src="http://static.funshion.com/open/static/img/logo.gif" height="70" alt="北京风行在线技术有限公司"/></a>
&nbsp;&nbsp;
<a href="http://www.kejishidai.cn"><img src="http://simg.ktvms.com/picture/logo.png" height="70" alt="北京可即时代网络公司"/></a>
<a href="https://www.kuaishou.com/"><img src="https://inews.gtimg.com/newsapp_bt/0/4400789257/1000" height="70" alt="快手短视频平台"/></a>

## 开源协议

eRPC 项目采用商业应用友好的 [Apache2.0](https://github.com/henrylee2cn/erpc/raw/master/LICENSE) 协议发布
