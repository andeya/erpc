# Teleport [![GitHub release](https://img.shields.io/github/release/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/releases) [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/teleport?style=flat-square)](http://goreportcard.com/report/henrylee2cn/teleport) [![github issues](https://img.shields.io/github/issues/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aopen+is%3Aissue) [![github closed issues](https://img.shields.io/github/issues-closed-raw/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aissue+is%3Aclosed) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/henrylee2cn/teleport) [![view examples](https://img.shields.io/badge/learn%20by-examples-00BCD4.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/tree/master/samples)


## 概述

Teleport是一个通用、高效、灵活的TCP Socket框架。

可用于Peer-Peer对等通信、RPC、长连接网关、微服务、推送服务，游戏服务等领域。

官方QQ群：Go-Web 编程 42730308    [![Go-Web 编程群](http://pub.idqqimg.com/wpa/images/group.png)](http://jq.qq.com/?_wv=1027&k=fzi4p1)

## 通信性能测试

- 测试环境

```
darwin amd64 4CPU 8GB
```

- teleport-socket

![tp_socket_benchmark](https://github.com/henrylee2cn/teleport/raw/develop/doc/tp_socket_benchmark.png)

**[test code](https://github.com/henrylee2cn/rpc-benchmark/tree/master/teleport)**

- 对比 rpcx

![rpcx_benchmark](https://github.com/henrylee2cn/teleport/raw/develop/doc/rpcx_benchmark.jpg)

**[test code](https://github.com/henrylee2cn/rpc-benchmark/tree/master/rpcx)**

- 火焰图 teleport-socket

![tp_socket_torch](https://github.com/henrylee2cn/teleport/raw/develop/doc/tp_socket_torch.png)

**[svg file](https://github.com/henrylee2cn/teleport/raw/develop/doc/tp_socket_torch.svg)**

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
- 支持推、拉、回复等通信模式
- 支持插件机制，可以自定义认证、心跳、微服务注册中心、统计信息插件等
- 无论服务器或客户端，均支持优雅重启、优雅关闭
- 支持实现反向代理功能
- 日志信息详尽，支持打印输入、输出消息的详细信息（状态码、消息头、消息体）
- 支持设置慢操作报警阈值
- 端点间通信使用I/O多路复用技术
- 支持设置读取包的大小限制（如果超出则断开连接）
- 提供Hander的上下文

## 4. 架构

### 4.1 名称解释

- **Peer：** 通信端点，可以是客户端或客户端
- **Session：** 连接会话，具有推、拉、回复、关闭等操作
- **Context：** 处理收到的或发送的数据包
- **Pull-Launch：** 从对端Peer拉数据
- **Pull-Handle：** 处理和回复对端Peer的拉请求
- **Push-Launch：** 将数据推送到对端Peer
- **Push-Handle：** 处理同伴的推送
- **Router：** Handler注册路由
- **Packet：** 数据包对应的结构体
- **Proto：** 数据包封包／解包的协议接口
- **Codec：** 用于`Packet.Body`的序列化工具
- **XferPipe：** 传输前对数据包数据进行一系列加工的管道
- **XferFilter：** 一个在数据包传输前，对数据进行加工的接口

### 4.2 执行层次

```
Peer -> Connection -> Socket -> Session -> Context
```

### 4.3 数据包内容

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

### 4.4 通信协议

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

- 创建一个Peer端点，服务端或客户端

```go
var cfg = &tp.PeerConfig{
    DefaultReadTimeout:  time.Minute * 5,
    DefaultWriteTimeout: time.Millisecond * 500,
    TlsCertFile:         "",
    TlsKeyFile:          "",
    SlowCometDuration:   time.Millisecond * 500,
    DefaultBodyCodec:     "json",
    PrintBody:           true,
    CountTime:           true,
    ListenAddrs: []string{
        "0.0.0.0:9090",
    },
}


var peer = tp.NewPeer(cfg)

// It can be used as a server
peer.Listen()

// It can also be used as a client at the same time
var sess, err = peer.Dial("127.0.0.1:8080")
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
func (h *Home) Test(args *[2]int) (int, *tp.Rerror) {
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
func UnknownPullHandle(ctx tp.UnknownPullCtx, body *[]byte) (interface{}, *tp.Rerror) {
    var v interface{}
    codecId, err := ctx.Unmarshal(*body, &v, true)
    if err != nil {
        return nil, tp.New*Rerror(0, err.Error())
    }
    tp.Infof("receive unknown pull:\n codec: %s\n content: %#v", codecId, v)
    return "this is reply string for unknown pull", nil
}

```

- 定义处理未知推送请求的操作

```go
func UnknownPushHandle(ctx tp.UnknownPushCtx, body *[]byte) {
    var v interface{}
    codecId, err := ctx.Unmarshal(*body, &v, true)
    if err != nil {
        tp.Errorf("%v", err)
    } else {
        tp.Infof("receive unknown push:\n codec: %s\n content: %#v", codecId, v)
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
    "encoding/json"
    "time"

    tp "github.com/henrylee2cn/teleport"
)

func main() {
    go tp.GraceSignal()
    // tp.SetReadLimit(10)
    tp.SetShutdown(time.Second*20, nil, nil)
    var cfg = &tp.PeerConfig{
        DefaultReadTimeout:  time.Minute * 5,
        DefaultWriteTimeout: time.Millisecond * 500,
        TlsCertFile:         "",
        TlsKeyFile:          "",
        SlowCometDuration:   time.Millisecond * 500,
        DefaultBodyCodec:    "json",
        PrintBody:           true,
        CountTime:           true,
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
func (h *Home) Test(args *map[string]interface{}) (map[string]interface{}, *tp.Rerror) {
    h.Session().Push("/push/test?tag=from home-test", map[string]interface{}{
        "your_id": h.Query().Get("peer_id"),
        "a":       1,
    })
    return map[string]interface{}{
        "your_args":   *args,
        "server_time": time.Now(),
    }, nil
}

func UnknownPullHandle(ctx tp.UnknownPullCtx) (interface{}, *tp.Rerror) {
    time.Sleep(1)
    var v = struct {
        ConnPort   int
        RawMessage json.RawMessage
        Bytes      []byte
    }{}
    codecId, err := ctx.Bind(&v)
    if err != nil {
        return nil, tp.NewRerror(1001, "bind error", err.Error())
    }
    tp.Debugf("UnknownPullHandle: codec: %d, conn_port: %d, RawMessage: %s, bytes: %s",
        codecId, v.ConnPort, v.RawMessage, v.Bytes,
    )
    return []string{"a", "aa", "aaa"}, nil
}
```

### client.go

```go
package main

import (
    "encoding/json"
    "time"

    tp "github.com/henrylee2cn/teleport"
)

func main() {
    go tp.GraceSignal()
    tp.SetShutdown(time.Second*20, nil, nil)
    var cfg = &tp.PeerConfig{
        DefaultReadTimeout:  time.Minute * 5,
        DefaultWriteTimeout: time.Millisecond * 500,
        TlsCertFile:         "",
        TlsKeyFile:          "",
        SlowCometDuration:   time.Millisecond * 500,
        DefaultBodyCodec:    "json",
        PrintBody:           true,
        CountTime:           true,
    }

    var peer = tp.NewPeer(cfg)
    defer peer.Close()
    peer.PushRouter.Reg(new(Push))

    {
        var sess, err = peer.Dial("127.0.0.1:9090")
        if err != nil {
            tp.Fatalf("%v", err)
        }

        var reply interface{}
        var pullcmd = sess.Pull(
            "/group/home/test?peer_id=client9090",
            map[string]interface{}{
                "conn_port": 9090,
                "bytes":     []byte("bytestest9090"),
            },
            &reply,
        )

        if pullcmd.Rerror() != nil {
            tp.Fatalf("pull error: %v", pullcmd.Rerror())
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
            struct {
                ConnPort   int
                RawMessage json.RawMessage
                Bytes      []byte
            }{
                9091,
                json.RawMessage(`{"RawMessage":"test9091"}`),
                []byte("bytes-test"),
            },
            &reply,
        )

        if pullcmd.Rerror() != nil {
            tp.Fatalf("pull error: %v", pullcmd.Rerror())
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
