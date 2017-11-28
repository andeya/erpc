# Teleport [![GitHub release](https://img.shields.io/github/release/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/releases) [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/teleport?style=flat-square)](http://goreportcard.com/report/henrylee2cn/teleport) [![github issues](https://img.shields.io/github/issues/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aopen+is%3Aissue) [![github closed issues](https://img.shields.io/github/issues-closed-raw/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aissue+is%3Aclosed) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/henrylee2cn/teleport) [![view examples](https://img.shields.io/badge/learn%20by-examples-00BCD4.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/tree/master/samples)

Teleport is a versatile, high-performance and flexible TCP socket framework.

It can be used for peer-peer, rpc, gateway, micro services, push services, game services and so on.

[简体中文](https://github.com/henrylee2cn/teleport/blob/master/README_ZH.md)


## Benchmark

- Test server configuration

```
darwin amd64 4CPU 8GB
```

- teleport-socket

![tp_socket_benchmark](https://github.com/henrylee2cn/teleport/raw/develop/doc/tp_socket_benchmark.png)

**[test code](https://github.com/henrylee2cn/rpc-benchmark/tree/master/teleport)**

- contrast rpcx

![rpcx_benchmark](https://github.com/henrylee2cn/teleport/raw/develop/doc/rpcx_benchmark.jpg)

**[test code](https://github.com/henrylee2cn/rpc-benchmark/tree/master/rpcx)**

- torch of teleport-socket

![tp_socket_torch](https://github.com/henrylee2cn/teleport/raw/develop/doc/tp_socket_torch.png)

**[svg file](https://github.com/henrylee2cn/teleport/raw/develop/doc/tp_socket_torch.svg)**

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

## 4. Architecture

### 4.1 Keywords

- **Peer:** A communication instance may be a client or a client
- **Session:** A connection session, with push, pull, reply, close and other methods of operation
- **Context:** Handle the received or send packets
- **Pull-Launch:** Pull data from the peer
- **Pull-Handle:** Handle and reply to the pull of peer
- **Push-Launch:** Push data to the peer
- **Push-Handle:** Handle the push of peer
- **Router:** Register handlers
- **Packet:** The corresponding structure of the data package
- **Proto:** The protocol interface of packet pack/unpack 
- **Codec:** Serialization interface for `Packet.Body`
- **XferPipe:** A series of pipelines to handle packet data before transfer
- **XferFilter:** A interface to handle packet data before transfer

### 4.2 Execution level

```
Peer -> Connection -> Socket -> Session -> Context
```

### 4.3 Packet

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

### 4.4 Protocol

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

- Create a server or client peer

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

// Also, it can also be used as a client at the same time
var sess, err = peer.Dial("127.0.0.1:8080")
if err != nil {
	tp.Panicf("%v", err)
}
```

- Define a controller and handler for pull request

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

- Define controller and handler for push request

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

- Define a handler for unknown pull request

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

- Define a handler for unknown push request

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

- Define a plugin

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

## 6. Demo

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

## 7. License

Teleport is under Apache v2 License. See the [LICENSE](https://github.com/henrylee2cn/teleport/raw/master/LICENSE) file for the full license text
