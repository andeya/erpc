# Teleport [![GitHub release](https://img.shields.io/github/release/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/releases) [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/teleport?style=flat-square)](http://goreportcard.com/report/henrylee2cn/teleport) [![github issues](https://img.shields.io/github/issues/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aopen+is%3Aissue) [![github closed issues](https://img.shields.io/github/issues-closed-raw/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aissue+is%3Aclosed) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/henrylee2cn/teleport) [![view examples](https://img.shields.io/badge/learn%20by-examples-00BCD4.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/tree/master/samples) [![view Go网络编程群](https://img.shields.io/badge/官方QQ群-Go网络编程(42730308)-27a5ea.svg?style=flat-square)](http://jq.qq.com/?_wv=1027&k=fzi4p1)


Teleport是一个通用、高效、灵活的Socket框架。

可用于Peer-Peer对等通信、RPC、长连接网关、微服务、推送服务，游戏服务等领域。


![Teleport-Architecture](https://github.com/henrylee2cn/teleport/raw/master/doc/teleport_architecture.png)


## 性能测试

**测试用例**

- 一个服务端与一个客户端进程，在同一台机器上运行
- CPU:    Intel Xeon E312xx (Sandy Bridge) 16 cores 2.53GHz
- Memory: 16G
- OS:     Linux 2.6.32-696.16.1.el6.centos.plus.x86_64, CentOS 6.4
- Go:     1.9.2
- 信息大小: 581 bytes
- 信息编码：protobuf
- 发送 1000000 条信息

**测试结果**

- teleport

并发client|平均值(ms)|中位数(ms)|最大值(ms)|最小值(ms)|吞吐率(TPS)
-------------|-------------|-------------|-------------|-------------|-------------
100|1|0|16|0|75505
500|9|11|97|0|52192
1000|19|24|187|0|50040
2000|39|54|409|0|42551
5000|96|128|1148|0|46367

- teleport/socket

并发client|平均值(ms)|中位数(ms)|最大值(ms)|最小值(ms)|吞吐率(TPS)
-------------|-------------|-------------|-------------|-------------|-------------
100|0|0|14|0|225682
500|2|1|24|0|212630
1000|4|3|51|0|180733
2000|8|6|64|0|183351
5000|21|18|651|0|133886

**[test code](https://github.com/henrylee2cn/rpc-benchmark/tree/master/teleport)**

- CPU火焰图 teleport/socket

![tp_socket_cpu_torch](https://github.com/henrylee2cn/teleport/raw/master/doc/tp_socket_torch.png)

**[svg file](https://github.com/henrylee2cn/teleport/raw/master/doc/tp_socket_torch.svg)**

## 1. 版本

版本 | 状态 | 分支
--------|--------|--------
v3      | release | [v3](https://github.com/henrylee2cn/teleport/tree/master)
v2      | release | [v2](https://github.com/henrylee2cn/teleport/tree/v2)
v1      | release | [v1](https://github.com/henrylee2cn/teleport/tree/v1)

## 2. 安装

```sh
go get -u github.com/henrylee2cn/teleport
```

## 3. 特性

- 服务器和客户端之间对等通信，两者API方法基本一致
- 支持定制通信协议
- 可设置底层套接字读写缓冲区的大小
- 底层通信数据包包含`Header`和`Body`两部分
- 支持单独定制`Header`和`Body`编码类型，例如`JSON` `Protobuf` `string`
- 数据包`Header`包含与HTTP header相同格式的元信息
- 支持推、拉、回复等通信方法
- 支持插件机制，可以自定义认证、心跳、微服务注册中心、统计信息插件等
- 无论服务器或客户端，均支持优雅重启、优雅关闭
- 支持实现反向代理功能
- 日志信息详尽，支持打印输入、输出消息的详细信息（状态码、消息头、消息体）
- 支持设置慢操作报警阈值
- 端点间通信使用I/O多路复用技术
- 支持设置读取包的大小限制（如果超出则断开连接）
- 提供Hander的上下文
- 客户端的Session支持断线后自动重连
- 支持的网络类型：`tcp`、`tcp4`、`tcp6`、`unix`、`unixpacket`等

## 4. 架构

### 4.1 名称解释

- **Peer：** 通信端点，可以是服务端或客户端
- **Socket：** 对net.Conn的封装，增加自定义包协议、传输管道等功能
- **Packet：** 数据包内容元素对应的结构体
- **Proto：** 数据包封包／解包的协议接口
- **Codec：** 用于`Packet.Body`的序列化工具
- **XferPipe：** 数据包字节流的编码处理管道，如压缩、加密、校验等
- **XferFilter：** 一个在数据包传输前，对数据进行加工的接口
- **Plugin：** 贯穿于通信各个环节的插件
- **Session：** 基于Socket封装的连接会话，提供的推、拉、回复、关闭等会话操作
- **Context：** 连接会话中一次通信（如PULL-REPLY, PUSH）的上下文对象
- **Pull-Launch：** 从对端Peer拉数据
- **Pull-Handle：** 处理和回复对端Peer的拉请求
- **Push-Launch：** 将数据推送到对端Peer
- **Push-Handle：** 处理同伴的推送
- **Router：** 通过请求信息（如URI）索引响应函数（Handler）的路由器


### 4.2 数据包内容

每个数据包的内容如下:

```go
// in socket package
type (
    // Packet a socket data packet.
    Packet struct {
        // packet sequence
        seq uint64
        // packet type, such as PULL, PUSH, REPLY
        ptype byte
        // URL string
        uri string
        // metadata
        meta *utils.Args
        // body codec type
        bodyCodec byte
        // body object
        body interface{}
        // newBodyFunc creates a new body by packet type and URI.
        // Note:
        //  only for writing packet;
        //  should be nil when reading packet.
        newBodyFunc NewBodyFunc
        // XferPipe transfer filter pipe, handlers from outer-most to inner-most.
        // Note: the length can not be bigger than 255!
        xferPipe *xfer.XferPipe
        // packet size
        size uint32
        next *Packet
    }

    // NewBodyFunc creates a new body by header info.
    NewBodyFunc func(seq uint64, ptype byte, uri string) interface{}
)

// in xfer package
type (
    // XferPipe transfer filter pipe, handlers from outer-most to inner-most.
    // Note: the length can not be bigger than 255!
    XferPipe struct {
        filters []XferFilter
    }
    // XferFilter handles byte stream of packet when transfer.
    XferFilter interface {
        Id() byte
        OnPack([]byte) ([]byte, error)
        OnUnpack([]byte) ([]byte, error)
    }
)
```

### 4.3 通信协议

支持通过接口定制自己的通信协议：

```go
type (
    // Proto pack/unpack protocol scheme of socket packet.
    Proto interface {
        // Version returns the protocol's id and name.
        Version() (byte, string)
        // Pack pack socket data packet.
        // Note: Make sure to write only once or there will be package contamination!
        Pack(*Packet) error
        // Unpack unpack socket data packet.
        // Note: Concurrent unsafe!
        Unpack(*Packet) error
    }
    ProtoFunc func(io.ReadWriter) Proto
)
```


接着，你可以使用以下任意方式指定自己的通信协议：

```go
func SetDefaultProtoFunc(socket.ProtoFunc)
func (*Peer) ServeConn(conn net.Conn, protoFunc ...socket.ProtoFunc) Session
func (*Peer) DialContext(ctx context.Context, addr string, protoFunc ...socket.ProtoFunc) (Session, *Rerror)
func (*Peer) Dial(addr string, protoFunc ...socket.ProtoFunc) (Session, *Rerror)
func (*Peer) Listen(protoFunc ...socket.ProtoFunc) error
```

## 5. 用法

- Peer端点（服务端或客户端）示例

```go
// Start a server
var peer1 = tp.NewPeer(tp.PeerConfig{
    ListenAddress: "0.0.0.0:9090", // for server role
})
peer1.Listen()

...

// Start a client
var peer2 = tp.NewPeer(tp.PeerConfig{})
var sess, err = peer2.Dial("127.0.0.1:8080")
```


- PullController模板示例

```go
type XxxPullController struct {
    tp.PullCtx
}
// XxZz register the route: /aaa/xx_zz
func (x *XxxPullController) XxZz(args *<T>) (<T>, *tp.Rerror) {
    ...
    return r, nil
}
// YyZz register the route: /aaa/yy_zz
func (x *XxxPullController) YyZz(args *<T>) (<T>, *tp.Rerror) {
    ...
    return r, nil
}
```

- PushController模板示例

```go
type XxxPushController struct {
    tp.PushCtx
}
// XxZz register the route: /bbb/yy_zz
func (b *XxxPushController) XxZz(args *<T>) *tp.Rerror {
    ...
    return r, nil
}
// YyZz register the route: /bbb/yy_zz
func (b *XxxPushController) YyZz(args *<T>) *tp.Rerror {
    ...
    return r, nil
}
```

- UnknownPullHandler模板示例

```go
func XxxUnknownPullHandler (ctx tp.UnknownPullCtx) (interface{}, *tp.Rerror) {
    ...
    return r, nil
}
```

- UnknownPushHandler模板示例

```go
func XxxUnknownPushHandler(ctx tp.UnknownPushCtx) *tp.Rerror {
    ...
    return nil
}
```

- 插件示例

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

// PostReadPullHeader converts the alias of this service.
func (p *AliasPlugin) PostReadPullHeader(ctx tp.ReadCtx) *tp.Rerror {
    var u = ctx.Input().Header.Uri
    if p.Aliases != nil {
        if a = p.Aliases[u]; a != "" {
            ctx.Input().Header.Uri = a
        }
    }
    return nil
}
```

- 注册以上操作和插件示例到路由

```go
aliasesPlugin := NewAliasPlugin()
aliasesPlugin.Alias("/alias", "/origin")

pullGroup := peer.PullRouter.Group("pull", aliasesPlugin)
pullGroup.Reg(new(XxxPullController))
peer.PullRouter.SetUnknown(XxxUnknownPullHandler)

pushGroup := peer.PushRouter.Group("push")
pushGroup.Reg(new(XxxPushController), aliasesPlugin)
peer.PushRouter.SetUnknown(XxxUnknownPushHandler)
```

## 6. 完整示例

### server.go

```go
package main

import (
    "encoding/json"
    "time"

    tp "github.com/henrylee2cn/teleport"
)

func main() {
    go tp.GraceSignal()
    // tp.SetReadLimit(10)
    tp.SetShutdown(time.Second*20, nil, nil)
    var peer = tp.NewPeer(tp.PeerConfig{
        SlowCometDuration: time.Millisecond * 500,
        PrintBody:         true,
        CountTime:         true,
        ListenAddress:     "0.0.0.0:9090",
    })
    group := peer.PullRouter.Group("group")
    group.Reg(new(Home))
    peer.PullRouter.SetUnknown(UnknownPullHandle)
    peer.Listen()
}

// Home controller
type Home struct {
    tp.PullCtx
}

// Test handler
func (h *Home) Test(args *map[string]interface{}) (map[string]interface{}, *tp.Rerror) {
    h.Session().Push("/push/test?tag=from home-test", map[string]interface{}{
        "your_id": h.Query().Get("peer_id"),
    })
    meta := h.CopyMeta()
    meta.VisitAll(func(k, v []byte) {
        tp.Infof("meta: key: %s, value: %s", k, v)
    })
    time.Sleep(5e9)
    return map[string]interface{}{
        "your_args":   *args,
        "server_time": time.Now(),
        "meta":        meta.String(),
    }, nil
}

func UnknownPullHandle(ctx tp.UnknownPullCtx) (interface{}, *tp.Rerror) {
    time.Sleep(1)
    var v = struct {
        RawMessage json.RawMessage
        Bytes      []byte
    }{}
    codecId, err := ctx.Bind(&v)
    if err != nil {
        return nil, tp.NewRerror(1001, "bind error", err.Error())
    }
    tp.Debugf("UnknownPullHandle: codec: %d, RawMessage: %s, bytes: %s",
        codecId, v.RawMessage, v.Bytes,
    )
    ctx.Session().Push("/push/test?tag=from home-test", map[string]interface{}{
        "your_id": ctx.Query().Get("peer_id"),
    })
    return map[string]interface{}{
        "your_args":   v,
        "server_time": time.Now(),
        "meta":        ctx.CopyMeta().String(),
    }, nil
}
```

### client.go

```go
package main

import (
    "encoding/json"
    "time"

    tp "github.com/henrylee2cn/teleport"
    "github.com/henrylee2cn/teleport/socket"
)

func main() {
    go tp.GraceSignal()
    tp.SetShutdown(time.Second*20, nil, nil)
    var peer = tp.NewPeer(tp.PeerConfig{
        SlowCometDuration: time.Millisecond * 500,
        // DefaultBodyCodec:    "json",
        PrintBody:   true,
        CountTime:   true,
        RedialTimes: 3,
    })
    defer peer.Close()
    peer.PushRouter.Reg(new(Push))

    var sess, err = peer.Dial("127.0.0.1:9090")
    if err != nil {
        tp.Fatalf("%v", err)
    }
    sess.SetId("testId")

    var reply interface{}
    for {
        var pullcmd = sess.Pull(
            "/group/home/test?peer_id=call-1",
            map[string]interface{}{
                "bytes": []byte("test bytes"),
            },
            &reply,
            socket.WithXferPipe('g'),
            socket.WithSetMeta("set", "0"),
            socket.WithAddMeta("add", "1"),
            socket.WithAddMeta("add", "2"),
        )
        if pullcmd.Rerror() != nil {
            tp.Errorf("pull error: %v", pullcmd.Rerror())
            time.Sleep(time.Second * 2)
        } else {
            break
        }
    }
    tp.Infof("test: %#v", reply)

    var pullcmd = sess.Pull(
        "/group/home/test_unknown?peer_id=call-2",
        struct {
            RawMessage json.RawMessage
            Bytes      []byte
        }{
            json.RawMessage(`{"RawMessage":"test_unknown"}`),
            []byte("test bytes"),
        },
        &reply,
        socket.WithXferPipe('g'),
    )

    if pullcmd.Rerror() != nil {
        tp.Fatalf("pull error: %v", pullcmd.Rerror())
    }
    tp.Infof("test_unknown: %#v", reply)
}

// Push controller
type Push struct {
    tp.PushCtx
}

// Test handler
func (p *Push) Test(args *map[string]interface{}) *tp.Rerror {
    tp.Infof("receive push(%s):\nargs: %#v\nquery: %#v\n", p.Ip(), args, p.Query())
    return nil
}
```

## 7. 扩展包

### 插件

package|import|description
----|------|-----------
[binder](https://github.com/henrylee2cn/tp-ext/blob/master/plugin-binder)|`import binder "github.com/henrylee2cn/tp-ext/plugin-binder"`|Parameter Binding Verification for Struct Handler
[heartbeat](https://github.com/henrylee2cn/tp-ext/blob/master/plugin-heartbeat)|`import heartbeat "github.com/henrylee2cn/tp-ext/plugin-heartbeat"`|A generic timing heartbeat plugin

[扩展库](https://github.com/henrylee2cn/tp-ext)

## 8. 基于Teleport的项目

project|description
----|---------------
[pholcus](https://github.com/henrylee2cn/pholcus)|Pholcus（幽灵蛛）是一款纯Go语言编写的支持分布式的高并发、重量级爬虫软件，定位于互联网数据采集，为具备一定Go或JS编程基础的人提供一个只需关注规则定制的功能强大的爬虫工具
[ants](https://github.com/henrylee2cn/ants)|Ants 是一套基于 Teleport 框架，类似于轻量级服务网格的微服务系统

## 9. 企业用户

[![深圳市梦之舵信息技术有限公司](https://statics.xiaoenai.com/v4/img/logo_zh.png)](http://www.xiaoenai.com)
&nbsp;&nbsp;
[![北京风行在线技术有限公司](http://static.funshion.com/open/static/img/logo.gif)](http://www.fun.tv)


## 10. 开源协议

Teleport 项目采用商业应用友好的 [Apache2.0](https://github.com/henrylee2cn/teleport/raw/master/LICENSE) 协议发布
