# Teleport [![GitHub release](https://img.shields.io/github/release/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/releases) [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/teleport?style=flat-square)](http://goreportcard.com/report/henrylee2cn/teleport) [![github issues](https://img.shields.io/github/issues/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aopen+is%3Aissue) [![github closed issues](https://img.shields.io/github/issues-closed-raw/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aissue+is%3Aclosed) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/henrylee2cn/teleport) [![view examples](https://img.shields.io/badge/learn%20by-examples-00BCD4.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/tree/master/samples) [![view Go网络编程群](https://img.shields.io/badge/官方QQ群-Go网络编程(42730308)-27a5ea.svg?style=flat-square)](http://jq.qq.com/?_wv=1027&k=fzi4p1)

Teleport is a versatile, high-performance and flexible socket framework.

It can be used for peer-peer, rpc, gateway, micro services, push services, game services and so on.

[简体中文](https://github.com/henrylee2cn/teleport/blob/master/README_ZH.md)


![Teleport-Architecture](https://github.com/henrylee2cn/teleport/raw/master/doc/teleport_architecture.png)


## Benchmark

**Test Case**

- A server and a client process, running on the same machine
- CPU:    Intel Xeon E312xx (Sandy Bridge) 16 cores 2.53GHz
- Memory: 16G
- OS:     Linux 2.6.32-696.16.1.el6.centos.plus.x86_64, CentOS 6.4
- Go:     1.9.2
- Message size: 581 bytes
- Message codec: protobuf
- Sent total 1000000 messages

**Test Results**

- teleport

client concurrency|mean(ms)|median(ms)|max(ms)|min(ms)|throughput(TPS)
-------------|-------------|-------------|-------------|-------------|-------------
100|1|0|16|0|75505
500|9|11|97|0|52192
1000|19|24|187|0|50040
2000|39|54|409|0|42551
5000|96|128|1148|0|46367

- teleport/socket

client concurrency|mean(ms)|median(ms)|max(ms)|min(ms)|throughput(TPS)
-------------|-------------|-------------|-------------|-------------|-------------
100|0|0|14|0|225682
500|2|1|24|0|212630
1000|4|3|51|0|180733
2000|8|6|64|0|183351
5000|21|18|651|0|133886

**[test code](https://github.com/henrylee2cn/rpc-benchmark/tree/master/teleport)**

- CPU torch of teleport/socket

![tp_socket_cpu_torch](https://github.com/henrylee2cn/teleport/raw/master/doc/tp_socket_torch.png)

**[svg file](https://github.com/henrylee2cn/teleport/raw/master/doc/tp_socket_torch.svg)**

## 1. Version

version | status | branch
--------|--------|--------
v3      | release | [v3](https://github.com/henrylee2cn/teleport/tree/master)
v2      | release | [v2](https://github.com/henrylee2cn/teleport/tree/v2)
v1      | release | [v1](https://github.com/henrylee2cn/teleport/tree/v1)


## 2. Install

```sh
go get -u github.com/henrylee2cn/teleport
```

## 3. Feature

- Server and client are peer-to-peer, have the same API method
- Support custom communication protocol
- Support set the size of socket I/O buffer
- Packet contains both Header and Body two parts
- Support for customizing head and body coding types separately, e.g `JSON` `Protobuf` `string`
- Packet Header contains metadata in the same format as http header
- Support push, pull, reply and other means of communication
- Support plug-in mechanism, can customize authentication, heartbeat, micro service registration center, statistics, etc.
- Whether server or client, the peer support reboot and shutdown gracefully
- Support reverse proxy
- Detailed log information, support print input and output details
- Supports setting slow operation alarm threshold
- Use I/O multiplexing technology
- Support setting the size of the reading packet (if exceed disconnect it)
- Provide the context of the handler
- Client session support automatically redials after disconnection
- Support network list: `tcp`, `tcp4`, `tcp6`, `unix`, `unixpacket` and so on

## 4. Architecture

### 4.1 Keywords

- **Peer:** A communication instance may be a server or a client
- **Socket:** Base on the net.Conn package, add custom package protocol, transfer pipelines and other functions
- **Packet:** The corresponding structure of the data package content element
- **Proto:** The protocol interface of packet pack/unpack 
- **Codec:** Serialization interface for `Packet.Body`
- **XferPipe:** Packet bytes encoding pipeline, such as compression, encryption, calibration and so on
- **XferFilter:** A interface to handle packet data before transfer
- **Plugin:** Plugins that cover all aspects of communication
- **Session:** A connection session, with push, pull, reply, close and other methods of operation
- **Context:** Handle the received or send packets
- **Pull-Launch:** Pull data from the peer
- **Pull-Handle:** Handle and reply to the pull of peer
- **Push-Launch:** Push data to the peer
- **Push-Handle:** Handle the push of peer
- **Router:** Router that route the response handler by request information(such as a URI)

### 4.2 Packet

The contents of every one packet:

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

### 4.3 Protocol

You can customize your own communication protocol by implementing the interface:

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

Next, you can specify the communication protocol in the following ways:

```go
func SetDefaultProtoFunc(socket.ProtoFunc)
func (*Peer) ServeConn(conn net.Conn, protoFunc ...socket.ProtoFunc) Session
func (*Peer) DialContext(ctx context.Context, addr string, protoFunc ...socket.ProtoFunc) (Session, *Rerror)
func (*Peer) Dial(addr string, protoFunc ...socket.ProtoFunc) (Session, *Rerror)
func (*Peer) Listen(protoFunc ...socket.ProtoFunc) error
```

## 5. Usage

- Peer(server or client) Demo

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

- PullController Model Demo

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

- PushController Model Demo

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

- UnknownPullHandler Type Demo

```go
func XxxUnknownPullHandler (ctx tp.UnknownPullCtx) (interface{}, *tp.Rerror) {
    ...
    return r, nil
}
```

- UnknownPushHandler Type Demo

```go
func XxxUnknownPushHandler(ctx tp.UnknownPushCtx) *tp.Rerror {
    ...
    return nil
}
```

- Plugin Demo

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

- Register above handler and plugin

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

## 6. Complete Demo

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

## 7. Extensions

### Plugin

package|import|description
----|------|-----------
[binder](https://github.com/henrylee2cn/tp-ext/blob/master/plugin-binder)|`import binder "github.com/henrylee2cn/tp-ext/plugin-binder"`|Parameter Binding Verification for Struct Handler

### Sundry

package|import|description
----|------|-----------
[heartbeat](https://github.com/henrylee2cn/tp-ext/blob/master/sundry-heartbeat)|`import heartbeat "github.com/henrylee2cn/tp-ext/sundry-heartbeat"`|A generic timing heartbeat package

[Extensions Repository](https://github.com/henrylee2cn/tp-ext)

## 8. Projects based on Teleport

project|description
----|---------------
[pholcus](https://github.com/henrylee2cn/pholcus)|Pholcus is a distributed, high concurrency and powerful web crawler software
[ants](https://github.com/henrylee2cn/ants)|Ants is a set of microservices-system based on Teleport framework and similar to lightweight service mesh

## 9. Business Users

[![深圳市梦之舵信息技术有限公司](https://statics.xiaoenai.com/v4/img/logo_zh.png)](http://www.xiaoenai.com)
&nbsp;&nbsp;
[![北京风行在线技术有限公司](http://static.funshion.com/open/static/img/logo.gif)](http://www.fun.tv)

## 10. License

Teleport is under Apache v2 License. See the [LICENSE](https://github.com/henrylee2cn/teleport/raw/master/LICENSE) file for the full license text
