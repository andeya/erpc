# Teleport [![GitHub release](https://img.shields.io/github/release/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/releases) [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/teleport?style=flat-square)](http://goreportcard.com/report/henrylee2cn/teleport) [![github issues](https://img.shields.io/github/issues/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aopen+is%3Aissue) [![github closed issues](https://img.shields.io/github/issues-closed-raw/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aissue+is%3Aclosed) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/henrylee2cn/teleport) [![view examples](https://img.shields.io/badge/learn%20by-examples-00BCD4.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/tree/v5/examples)
<!-- [![view Go网络编程群](https://img.shields.io/badge/官方QQ群-Go网络编程(42730308)-27a5ea.svg?style=flat-square)](http://jq.qq.com/?_wv=1027&k=fzi4p1) -->


Teleport是一个通用、高效、灵活的Socket框架。

可用于Peer-Peer对等通信、RPC、长连接网关、微服务、推送服务，游戏服务等领域。


![Teleport-Framework](https://github.com/henrylee2cn/teleport/raw/v5/doc/teleport_module_diagram.png)


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

- teleport

| 并发client | 平均值(ms) | 中位数(ms) | 最大值(ms) | 最小值(ms) | 吞吐率(TPS) |
| -------- | ------- | ------- | ------- | ------- | -------- |
| 100      | 1       | 0       | 16      | 0       | 75505    |
| 500      | 9       | 11      | 97      | 0       | 52192    |
| 1000     | 19      | 24      | 187     | 0       | 50040    |
| 2000     | 39      | 54      | 409     | 0       | 42551    |
| 5000     | 96      | 128     | 1148    | 0       | 46367    |

- teleport/socket

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

- CPU耗时火焰图 teleport/socket

![tp_socket_profile_torch](https://github.com/henrylee2cn/teleport/raw/v5/doc/tp_socket_profile_torch.png)

**[svg file](https://github.com/henrylee2cn/teleport/raw/v5/doc/tp_socket_profile_torch.svg)**

- 堆栈信息火焰图 teleport/socket

![tp_socket_heap_torch](https://github.com/henrylee2cn/teleport/raw/v5/doc/tp_socket_heap_torch.png)

**[svg file](https://github.com/henrylee2cn/teleport/raw/v5/doc/tp_socket_heap_torch.svg)**


## 版本

| 版本   | 状态      | 分支                                       |
| ---- | ------- | ---------------------------------------- |
| v6      | release | [master](https://github.com/henrylee2cn/teleport/tree/master) |
| v5      | release | [v5](https://github.com/henrylee2cn/teleport/tree/v5) |
| v4      | release | [v4](https://github.com/henrylee2cn/teleport/tree/v4) |
| v3      | release | [v3](https://github.com/henrylee2cn/teleport/tree/v3) |
| v2      | release | [v2](https://github.com/henrylee2cn/teleport/tree/v2) |
| v1      | release | [v1](https://github.com/henrylee2cn/teleport/tree/v1) |

## 安装

```sh
go get -u -f github.com/henrylee2cn/teleport
```

## 特性

- 服务器和客户端之间对等通信，两者API方法基本一致
- 支持定制通信协议
- 可设置底层套接字读写缓冲区的大小
- 底层通信数据包包含`Header`和`Body`两部分
- 数据包`Header`包含与HTTP header相同格式的元信息
- 支持单独定制`Body`编码类型，例如`JSON` `Protobuf` `string`
- 支持推、拉、回复等通信方法
- 支持插件机制，可以自定义认证、心跳、微服务注册中心、统计信息插件等
- 无论服务器或客户端，均支持优雅重启、优雅关闭
- 支持实现反向代理功能
- 日志信息详尽，支持打印输入、输出报文的详细信息（状态码、头信息、正文）
- 支持设置慢操作报警阈值
- 端点间通信使用I/O多路复用技术
- 支持设置读取包的大小限制（如果超出则断开连接）
- 提供Handler的上下文
- 客户端的Session支持断线后自动重连
- 提供对连接文件描述符（fd）的操作接口
- 支持的网络类型：
    - `tcp`
    - `tcp4`
    - `tcp6`
    - `unix`
    - `unixpacket`
    - `quic`

## 代码示例

### server.go

```go
package main

import (
	"fmt"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	// graceful
	go tp.GraceSignal()

	// server peer
	srv := tp.NewPeer(tp.PeerConfig{
		CountTime:   true,
		ListenPort:  9090,
		PrintDetail: true,
	})

	// router
	srv.RouteCall(new(Math))

	// broadcast per 5s
	go func() {
		for {
			time.Sleep(time.Second * 5)
			srv.RangeSession(func(sess tp.Session) bool {
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
	tp.CallCtx
}

// Add handles addition request
func (m *Math) Add(arg *[]int) (int, *tp.Status) {
	// test query parameter
	tp.Infof("author: %s", m.Query().Get("author"))
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

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	// log level
	tp.SetLoggerLevel("ERROR")

	cli := tp.NewPeer(tp.PeerConfig{})
	defer cli.Close()

	cli.RoutePush(new(Push))

	sess, stat := cli.Dial(":9090")
	if !stat.OK() {
		tp.Fatalf("%v", stat)
	}

	var result int
	stat = sess.Call("/math/add?author=henrylee2cn",
		[]int{1, 2, 3, 4, 5},
		&result,
	).Status()
	if !stat.OK() {
		tp.Fatalf("%v", stat)
	}
	tp.Printf("result: %d", result)

	tp.Printf("wait for 10s...")
	time.Sleep(time.Second * 10)
}

// Push push handler
type Push struct {
	tp.PushCtx
}

// Push handles '/push/status' message
func (p *Push) Status(arg *string) *tp.Status {
	tp.Printf("%s", *arg)
	return nil
}
```

[更多示例](https://github.com/henrylee2cn/teleport/blob/master/examples)


## 框架设计

### 名称解释

- **Peer：** 通信端点，可以是服务端或客户端
- **Socket：** 对net.Conn的封装，增加自定义包协议、传输管道等功能
- *Message：** 数据包内容元素对应的结构体
- **Proto：** 数据包封包／解包的协议接口
- **Codec：** 用于`Body`的序列化工具
- **XferPipe：** 数据包字节流的编码处理管道，如压缩、加密、校验等
- **XferFilter：** 一个在数据包传输前，对数据进行加工的接口
- **Plugin：** 贯穿于通信各个环节的插件
- **Session：** 基于Socket封装的连接会话，提供的推、拉、回复、关闭等会话操作
- **Context：** 连接会话中一次通信（如PULL-REPLY, PUSH）的上下文对象
- **Call-Launch：** 从对端Peer拉数据
- **Call-Handle：** 处理和回复对端Peer的拉请求
- **Push-Launch：** 将数据推送到对端Peer
- **Push-Handle：** 处理同伴的推送
- **Router：** 通过请求信息（如URI）索引响应函数（Handler）的路由器


### 数据报文

抽象应用层的数据报文（Message 对象）并与 HTTP 报文兼容：

![tp_data_message](https://github.com/henrylee2cn/teleport/raw/v5/doc/tp_data_message.png)


### 通信协议

支持通过接口定制自己的通信协议：

```go
type (
    // Proto pack/unpack protocol scheme of socket message.
    Proto interface {
        // Version returns the protocol's id and name.
        Version() (byte, string)
        // Pack writes the Message into the connection.
        // NOTE: Make sure to write only once or there will be package contamination!
        Pack(Message) error
        // Unpack reads bytes from the connection to the Message.
        // NOTE: Concurrent unsafe!
        Unpack(Message) error
    }
    ProtoFunc func(io.ReadWriter) Proto
)
```


接着，你可以使用以下任意方式指定自己的通信协议：

```go
func SetDefaultProtoFunc(ProtoFunc)
type Peer interface {
    ...
    ServeConn(conn net.Conn, protoFunc ...ProtoFunc) Session
    DialContext(ctx context.Context, addr string, protoFunc ...ProtoFunc) (Session, *Status)
    Dial(addr string, protoFunc ...ProtoFunc) (Session, *Status)
    Listen(protoFunc ...ProtoFunc) error
    ...
}
```

默认的协议`RawProto`(Big Endian)：

```sh
{4 bytes message length}
{1 byte protocol version}
{1 byte transfer pipe length}
{transfer pipe IDs}
# The following is handled data by transfer pipe
{1 bytes sequence length}
{sequence (HEX 36 string of int32)}
{1 byte message type} # e.g. CALL:1; REPLY:2; PUSH:3
{1 bytes service method length}
{service method}
{2 bytes metadata length}
{metadata(urlencoded)}
{1 byte body codec id}
{body}
```


### 过滤管道

传输数据的过滤管道。
```go
// XferFilter handles byte stream of message when transfer.
type XferFilter interface {
    // ID returns transfer filter id.
    ID() byte
    // Name returns transfer filter name.
    Name() string
    // OnPack performs filtering on packing.
    OnPack([]byte) ([]byte, error)
    // OnUnpack performs filtering on unpacking.
    OnUnpack([]byte) ([]byte, error)
}
// Get returns transfer filter by id.
func Get(id byte) (XferFilter, error)
// GetByName returns transfer filter by name.
func GetByName(name string) (XferFilter, error)

// XferPipe transfer filter pipe, handlers from outer-most to inner-most.
// NOTE: the length can not be bigger than 255!
type XferPipe struct {
    // Has unexported fields.
}
func NewXferPipe() *XferPipe
func (x *XferPipe) Append(filterID ...byte) error
func (x *XferPipe) AppendFrom(src *XferPipe)
func (x *XferPipe) IDs() []byte
func (x *XferPipe) Len() int
func (x *XferPipe) Names() []string
func (x *XferPipe) OnPack(data []byte) ([]byte, error)
func (x *XferPipe) OnUnpack(data []byte) ([]byte, error)
func (x *XferPipe) Range(callback func(idx int, filter XferFilter) bool)
func (x *XferPipe) Reset()
```


### 编解码器

数据包中Body内容的编解码器。

```go
type Codec interface {
    // ID returns codec id.
    ID() byte
    // Name returns codec name.
    Name() string
    // Marshal returns the encoding of v.
    Marshal(v interface{}) ([]byte, error)
    // Unmarshal parses the encoded data and stores the result
    // in the value pointed to by v.
    Unmarshal(data []byte, v interface{}) error
}
```


### 插件

运行过程中以挂载方式执行的插件。

```go
type (
    // Plugin plugin background
    Plugin interface {
        Name() string
    }
    // PreNewPeerPlugin is executed before creating peer.
    PreNewPeerPlugin interface {
        Plugin
        PreNewPeer(*PeerConfig, *PluginContainer) error
    }
    ...
)
```


## 用法

### Peer端点（服务端或客户端）示例

```go
// Start a server
var peer1 = tp.NewPeer(tp.PeerConfig{
    ListenPort: 9090, // for server role
})
peer1.Listen()

...

// Start a client
var peer2 = tp.NewPeer(tp.PeerConfig{})
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
    tp.SetServiceMethodMapper(tp.HTTPServiceMethodMapper)
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
    tp.SetServiceMethodMapper(tp.RPCServiceMethodMapper)
    ```

### Call-Struct 接口模版

```go
type Aaa struct {
    tp.CallCtx
}
func (x *Aaa) XxZz(arg *<T>) (<T>, *tp.Status) {
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
func XxZz(ctx tp.CallCtx, arg *<T>) (<T>, *tp.Status) {
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
    tp.PushCtx
}
func (b *Bbb) YyZz(arg *<T>) *tp.Status {
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
func YyZz(ctx tp.PushCtx, arg *<T>) *tp.Status {
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
func XxxUnknownCall (ctx tp.UnknownCallCtx) (interface{}, *tp.Status) {
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
func XxxUnknownPush(ctx tp.UnknownPushCtx) *tp.Status {
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
    _ tp.PostReadCallHeaderPlugin = new(ignoreCase)
    _ tp.PostReadPushHeaderPlugin = new(ignoreCase)
)

func (i *ignoreCase) Name() string {
    return "ignoreCase"
}

func (i *ignoreCase) PostReadCallHeader(ctx tp.ReadCtx) *tp.Status {
    // Dynamic transformation path is lowercase
    ctx.UriObject().Path = strings.ToLower(ctx.UriObject().Path)
    return nil
}

func (i *ignoreCase) PostReadPushHeader(ctx tp.ReadCtx) *tp.Status {
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
    DefaultDialTimeout time.Duration `yaml:"default_dial_timeout" ini:"default_dial_timeout" comment:"Default maximum duration for dialing; for client role; ns,µs,ms,s,m,h"`
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
| [json](https://github.com/henrylee2cn/teleport/blob/v5/codec/json_codec.go) | `import "github.com/henrylee2cn/teleport/codec"` | JSON codec(teleport own)     |
| [protobuf](https://github.com/henrylee2cn/teleport/blob/v5/codec/protobuf_codec.go) | `import "github.com/henrylee2cn/teleport/codec"` | Protobuf codec(teleport own) |
| [thrift](https://github.com/henrylee2cn/teleport/blob/v5/codec/thrift_codec.go) | `import "github.com/henrylee2cn/teleport/codec"` | Form(url encode) codec(teleport own)   |
| [xml](https://github.com/henrylee2cn/teleport/blob/v5/codec/xml_codec.go) | `import "github.com/henrylee2cn/teleport/codec"` | Form(url encode) codec(teleport own)   |
| [plain](https://github.com/henrylee2cn/teleport/blob/v5/codec/plain_codec.go) | `import "github.com/henrylee2cn/teleport/codec"` | Plain text codec(teleport own)   |
| [form](https://github.com/henrylee2cn/teleport/blob/v5/codec/form_codec.go) | `import "github.com/henrylee2cn/teleport/codec"` | Form(url encode) codec(teleport own)   |

### 插件

| package                                  | import                                   | description                              |
| ---------------------------------------- | ---------------------------------------- | ---------------------------------------- |
| [auth](https://github.com/henrylee2cn/teleport/tree/v5/plugin/auth) | `import "github.com/henrylee2cn/teleport/plugin/auth"` | A auth plugin for verifying peer at the first time |
| [binder](https://github.com/henrylee2cn/teleport/tree/v5/plugin/binder) | `import binder "github.com/henrylee2cn/teleport/plugin/binder"` | Parameter Binding Verification for Struct Handler |
| [heartbeat](https://github.com/henrylee2cn/teleport/tree/v5/plugin/heartbeat) | `import heartbeat "github.com/henrylee2cn/teleport/plugin/heartbeat"` | A generic timing heartbeat plugin        |
| [proxy](https://github.com/henrylee2cn/teleport/tree/v5/plugin/proxy) | `import "github.com/henrylee2cn/teleport/plugin/proxy"` | A proxy plugin for handling unknown calling or pushing |
[secure](https://github.com/henrylee2cn/teleport/tree/v5/plugin/secure)|`import secure "github.com/henrylee2cn/teleport/plugin/secure"`|Encrypting/decrypting the message body

### 协议

| package                                  | import                                   | description                              |
| ---------------------------------------- | ---------------------------------------- | ---------------------------------------- |
| [rawproto](https://github.com/henrylee2cn/teleport/tree/v5/proto/rawproto) | `import "github.com/henrylee2cn/teleport/proto/rawproto` | 一个高性能的通信协议（teleport默认）|
| [jsonproto](https://github.com/henrylee2cn/teleport/tree/v5/proto/jsonproto) | `import "github.com/henrylee2cn/teleport/proto/jsonproto"` | JSON 格式的通信协议     |
| [pbproto](https://github.com/henrylee2cn/teleport/tree/v5/proto/pbproto) | `import "github.com/henrylee2cn/teleport/proto/pbproto"` | Protobuf 格式的通信协议     |
| [thriftproto](https://github.com/henrylee2cn/teleport/tree/v5/proto/thriftproto) | `import "github.com/henrylee2cn/teleport/proto/thriftproto"` | Thrift 格式的通信协议     |
| [httproto](https://github.com/henrylee2cn/teleport/tree/v5/proto/httproto) | `import "github.com/henrylee2cn/teleport/proto/httproto"` | HTTP 格式的通信协议     |

### 传输过滤器

| package                                  | import                                   | description                              |
| ---------------------------------------- | ---------------------------------------- | ---------------------------------------- |
| [gzip](https://github.com/henrylee2cn/teleport/tree/v5/xfer/gzip) | `import "github.com/henrylee2cn/teleport/xfer/gzip"` | Gzip(teleport own)                       |
| [md5](https://github.com/henrylee2cn/teleport/tree/v5/xfer/md5) | `import "github.com/henrylee2cn/teleport/xfer/md5"` | Provides a integrity check transfer filter |

### 其他模块

| package                                  | import                                   | description                              |
| ---------------------------------------- | ---------------------------------------- | ---------------------------------------- |
| [multiclient](https://github.com/henrylee2cn/teleport/tree/v5/mixer/multiclient) | `import "github.com/henrylee2cn/teleport/mixer/multiclient"` | Higher throughput client connection pool when transferring large messages (such as downloading files) |
| [websocket](https://github.com/henrylee2cn/teleport/tree/v5/mixer/websocket) | `import "github.com/henrylee2cn/teleport/mixer/websocket"` | Makes the Teleport framework compatible with websocket protocol as specified in RFC 6455 |
| [evio](https://github.com/henrylee2cn/teleport/tree/v5/mixer/evio) | `import "github.com/henrylee2cn/teleport/mixer/evio"` | A fast event-loop networking framework that uses the teleport API layer |
| [html](https://github.com/xiaoenai/tp-micro/tree/master/helper/mod-html) | `html "github.com/xiaoenai/tp-micro/helper/mod-html"` | HTML render for http client |

## 基于Teleport的项目

| project                                  | description                              |
| ---------------------------------------- | ---------------------------------------- |
| [TP-Micro](https://github.com/xiaoenai/tp-micro) | TP-Micro 是一个基于 Teleport 定制的、简约而强大的微服务框架          |
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

Teleport 项目采用商业应用友好的 [Apache2.0](https://github.com/henrylee2cn/teleport/raw/v5/LICENSE) 协议发布
