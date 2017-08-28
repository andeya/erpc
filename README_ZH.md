# Teleport [![GitHub release](https://img.shields.io/github/release/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/releases) [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/teleport?style=flat-square)](http://goreportcard.com/report/henrylee2cn/teleport) [![github issues](https://img.shields.io/github/issues/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aopen+is%3Aissue) [![github closed issues](https://img.shields.io/github/issues-closed-raw/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aissue+is%3Aclosed) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/henrylee2cn/teleport) [![view examples](https://img.shields.io/badge/learn%20by-examples-00BCD4.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/tree/master/samples)


## 概述

Teleport是一个通用、高效、灵活的TCP Socket框架。

可用于Peer-Peer对等通信、RPC、长连接网关、微服务、推送服务，游戏服务等领域。

官方QQ群：Go-Web 编程 42730308    [![Go-Web 编程群](http://pub.idqqimg.com/wpa/images/group.png)](http://jq.qq.com/?_wv=1027&k=fzi4p1)

![teleport_server](https://github.com/henrylee2cn/teleport/raw/master/doc/teleport_server.png)
<br>

*AB Testing 1: [Mac 4CPU 8GB] [single-process single-conn] teleport: QPS 39414*
![teleport_frame_client_ab_test](https://github.com/henrylee2cn/teleport/raw/master/doc/frame_client_ab.png)
<br>

*AB Testing 2: [Mac 4CPU 8GB] [single-process single-conn] teleport/socket: QPS 55419*
![teleport_socket_client_ab_test](https://github.com/henrylee2cn/teleport/raw/master/doc/socket_client_ab.png)

## 1. 版本

版本 | 状态 | 分支
--------|--------|--------
v2      | release | [master](https://github.com/henrylee2cn/teleport/tree/master)
v1      | release | [v1](https://github.com/henrylee2cn/teleport/tree/v1)


## 2. 安装

```sh
go get -u github.com/henrylee2cn/teleport
```

## 3. 特性

- 服务器和客户端之间对等通信，两者API方法基本一致
- 底层通信数据包包含`Header`和`Body`两部分
- 支持单独定制`Header`和`Body`编码类型，例如`JSON` `Protobuf`
- `Body`支持gzip压缩
- `Header`包含状态码及其描述文本
- 支持推、拉、回复等通信模式
- 支持插件机制，可以自定义认证、心跳、微服务注册中心、统计信息插件等
- 无论服务器或客户端，均支持都优雅重启、优雅关闭
- 支持实现反向代理功能
- 日志信息详尽，支持打印输入、输出消息的详细信息（状态码、消息头、消息体）
- 支持设置慢操作报警阈值
- 底层连接使用I/O缓冲区
- 端点间通信使用I/O多路复用技术

## 4. 架构

### 4.1 名称解释

- **Peer：**通信端点，可以是客户端或客户端
- **Session：**连接会话，具有推、拉、回复、关闭等操作
- **Context：**处理收到的或发送的数据包
- **Pull-Launch：**从对端Peer拉数据
- **Pull-Handle：**处理和回复对端Peer的拉请求
- **Push-Launch：**将数据推送到对端Peer
- **Push-Handle：**处理同伴的推送
- **Router：**Handler注册路由

### 4.2 执行层次

```
Peer -> Connection -> Socket -> Session -> Context
```


### 4.3 数据包

```
HeaderLength | HeaderCodecId | Header | BodyLength | BodyCodecId | Body
```

**注意：**

- HeaderLength: uint32, 4 bytes, big endian
- BodyLength: uint32, 4 bytes, big endian
- HeaderCodecId: uint8, 1 byte
- BodyCodecId: uint8, 1 byte

```go
type Packet struct {
    // HeaderCodec header codec name
    HeaderCodec string `json:"header_codec"`
    // BodyCodec body codec name
    BodyCodec string `json:"body_codec"`
    // header content
    Header *Header `json:"header"`
    // body content
    Body interface{} `json:"body"`
    // header length
    HeaderLength int64 `json:"header_length"`
    // body length
    BodyLength int64 `json:"body_length"`
    // HeaderLength + BodyLength
    Length int64 `json:"length"`
}
```

### 4.4 头信息

```go
type Header struct {
    // Packet id
    Id string
    // Service type
    Type int32
    // Service URI
    Uri string
    // Body encoding type
    Gzip int32
    // As reply, it indicates the service status code
    StatusCode int32
    // As reply, it indicates the service status text
    Status string
}
```

## 5. 用法

- 创建一个Peer端点，服务端或客户端

```go
var cfg = &tp.PeerConfig{
    DefaultReadTimeout:   time.Minute * 3,
    DefaultWriteTimeout:  time.Minute * 3,
    TlsCertFile:          "",
    TlsKeyFile:           "",
    SlowCometDuration:    time.Millisecond * 500,
    DefaultHeaderCodec:   "protobuf",
    DefaultBodyCodec:     "json",
    DefaultBodyGzipLevel: 5,
    PrintBody:            true,
    DefaultDialTimeout:   time.Second * 10,
    ListenAddrs: []string{
        "0.0.0.0:9090",
    },
}


var peer = tp.NewPeer(cfg)

// It can be used as a server
peer.Listen()

// It can also be used as a client at the same time
var sess, err = peer.Dial("127.0.0.1:8080", "peerid-client")
if err != nil {
    tp.Panicf("%v", err)
}
```

- 定义处理拉取请求的控制器（命名空间）及其操作

```go
// Home controller
type Home struct {
    tp.PullCtx
}

// Test handler
func (h *Home) Test(args *[2]int) (int, tp.Xerror) {
    a := (*args)[0]
    b := (*args)[1]
    return a + b, nil
}
```

- 定义处理推送请求的控制器（命名空间）及其操作

```go
// Msg controller
type Msg struct {
    tp.PushCtx
}

// Test handler
func (m *Msg) Test(args *map[string]interface{}) {
    tp.Infof("receive push(%s):\nargs: %#v\nquery: %#v\n", m.Ip(), args, m.Query())
}
```

- 定义处理未知拉取请求的操作

```go
func UnknownPullHandle(ctx tp.UnknownPullCtx, body *[]byte) (interface{}, tp.Xerror) {
    var v interface{}
    codecName, err := ctx.Unmarshal(*body, &v, true)
    if err != nil {
        return nil, tp.NewXerror(0, err.Error())
    }
    tp.Infof("receive unknown pull:\n codec: %s\n content: %#v", codecName, v)
    return "this is reply string for unknown pull", nil
}

```

- 定义处理未知推送请求的操作

```go
func UnknownPushHandle(ctx tp.UnknownPushCtx, body *[]byte) {
    var v interface{}
    codecName, err := ctx.Unmarshal(*body, &v, true)
    if err != nil {
        tp.Errorf("%v", err)
    } else {
        tp.Infof("receive unknown push:\n codec: %s\n content: %#v", codecName, v)
    }
}
```

- 定义插件

```go
// AliasPlugin can be used to set aliases for pull or push services
type AliasPlugin struct {
    Aliases map[string]string
}

// NewAliasPlugin creates a new NewAliasPlugin
func NewAliasPlugin() *AliasPlugin {
    return &AliasPlugin{Aliases: make(map[string]string)}
}

// Alias sets a alias for the uri.
// For example Alias("/arith/mul", "/mul")
func (p *AliasPlugin) Alias(alias string, uri string) {
    p.Aliases[alias] = uri
}

// Name return name of this plugin.
func (p *AliasPlugin) Name() string {
    return "AliasPlugin"
}

// PostReadHeader converts the alias of this service.
func (p *AliasPlugin) PostReadHeader(ctx tp.ReadCtx) error {
    var u = ctx.Input().Header.Uri
    if p.Aliases != nil {
        if a = p.Aliases[u]; a != "" {
            ctx.Input().Header.Uri = a
        }
    }
    return nil
}
```

- 注册以上操作和插件到路由

```go
aliasesPlugin := NewAliasPlugin()
aliasesPlugin.Alias("/alias", "/origin")
{
    pullGroup := peer.PullRouter.Group("pull", aliasesPlugin)
    pullGroup.Reg(new(Home))
    peer.PullRouter.SetUnknown(UnknownPullHandle)
}
{
    pushGroup := peer.PushRouter.Group("push")
    pushGroup.Reg(new(Msg), aliasesPlugin)
    peer.PushRouter.SetUnknown(UnknownPushHandle)
}
```

## 6. 示例

### server.go

```go
package main

import (
    "time"

    tp "github.com/henrylee2cn/teleport"
)

func main() {
    go tp.GraceSignal()
    tp.SetShutdown(time.Second*20, nil, nil)
    var cfg = &tp.PeerConfig{
        DefaultReadTimeout:   time.Minute * 3,
        DefaultWriteTimeout:  time.Minute * 3,
        TlsCertFile:          "",
        TlsKeyFile:           "",
        SlowCometDuration:    time.Millisecond * 500,
        DefaultHeaderCodec:   "protobuf",
        DefaultBodyCodec:     "json",
        DefaultBodyGzipLevel: 5,
        PrintBody:            true,
        ListenAddrs: []string{
            "0.0.0.0:9090",
            "0.0.0.0:9091",
        },
    }
    var peer = tp.NewPeer(cfg)
    {
        group := peer.PullRouter.Group("group")
        group.Reg(new(Home))
    }
    peer.PullRouter.SetUnknown(UnknownPullHandle)
    peer.Listen()
}

// Home controller
type Home struct {
    tp.PullCtx
}

// Test handler
func (h *Home) Test(args *map[string]interface{}) (map[string]interface{}, tp.Xerror) {
    h.Session().Push("/push/test?tag=from home-test", map[string]interface{}{
        "your_id": h.Query().Get("peer_id"),
        "a":       1,
    })
    return map[string]interface{}{
        "your_args":   *args,
        "server_time": time.Now(),
    }, nil
}

func UnknownPullHandle(ctx tp.UnknownPullCtx, body *[]byte) (interface{}, tp.Xerror) {
    var v interface{}
    codecName, err := ctx.Unmarshal(*body, &v, true)
    if err != nil {
        return nil, tp.NewXerror(0, err.Error())
    }
    tp.Debugf("unmarshal body: codec: %s, content: %#v", codecName, v)
    return []string{"a", "aa", "aaa"}, nil
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
    go tp.GraceSignal()
    tp.SetShutdown(time.Second*20, nil, nil)
    var cfg = &tp.PeerConfig{
        DefaultReadTimeout:   time.Minute * 3,
        DefaultWriteTimeout:  time.Minute * 3,
        TlsCertFile:          "",
        TlsKeyFile:           "",
        SlowCometDuration:    time.Millisecond * 500,
        DefaultHeaderCodec:   "protobuf",
        DefaultBodyCodec:     "json",
        DefaultBodyGzipLevel: 5,
        PrintBody:            false,
    }

    var peer = tp.NewPeer(cfg)
    peer.PushRouter.Reg(new(Push))

    {
        var sess, err = peer.Dial("127.0.0.1:9090", "simple_server:9090")
        if err != nil {
            tp.Panicf("%v", err)
        }

        var reply interface{}
        var pullcmd = sess.Pull(
            "/group/home/test?peer_id=client9090",
            map[string]interface{}{"conn_port": 9090},
            &reply,
        )

        if pullcmd.Xerror != nil {
            tp.Fatalf("pull error: %v", pullcmd.Xerror.Error())
        }
        tp.Infof("9090reply: %#v", reply)
    }

    {
        var sess, err = peer.Dial("127.0.0.1:9091")
        if err != nil {
            tp.Panicf("%v", err)
        }

        var reply interface{}
        var pullcmd = sess.Pull(
            "/group/home/test_unknown?peer_id=client9091",
            map[string]interface{}{"conn_port": 9091},
            &reply,
        )

        if pullcmd.Xerror != nil {
            tp.Fatalf("pull error: %v", pullcmd.Xerror.Error())
        }
        tp.Infof("9091reply test_unknown: %#v", reply)
    }
}

// Push controller
type Push struct {
    tp.PushCtx
}

// Test handler
func (p *Push) Test(args *map[string]interface{}) {
    tp.Infof("receive push(%s):\nargs: %#v\nquery: %#v\n", p.Ip(), args, p.Query())
}
```

## 7. 开源协议

Teleport 项目采用商业应用友好的 [Apache2.0](https://github.com/henrylee2cn/teleport/raw/master/LICENSE) 协议发布
