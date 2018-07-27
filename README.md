# Teleport [![GitHub release](https://img.shields.io/github/release/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/releases) [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/teleport?style=flat-square)](http://goreportcard.com/report/henrylee2cn/teleport) [![github issues](https://img.shields.io/github/issues/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aopen+is%3Aissue) [![github closed issues](https://img.shields.io/github/issues-closed-raw/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aissue+is%3Aclosed) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/henrylee2cn/teleport) [![view examples](https://img.shields.io/badge/learn%20by-examples-00BCD4.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/tree/v4/examples)
<!-- [![view Go网络编程群](https://img.shields.io/badge/官方QQ群-Go网络编程(42730308)-27a5ea.svg?style=flat-square)](http://jq.qq.com/?_wv=1027&k=fzi4p1) -->

Teleport is a versatile, high-performance and flexible socket framework.

It can be used for peer-peer, rpc, gateway, micro services, push services, game services and so on.

[简体中文](https://github.com/henrylee2cn/teleport/tree/v4/README_ZH.md)


![Teleport-Framework](https://github.com/henrylee2cn/teleport/raw/v4/doc/teleport_framework.png)


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

| client concurrency | mean(ms) | median(ms) | max(ms) | min(ms) | throughput(TPS) |
| ------------------ | -------- | ---------- | ------- | ------- | --------------- |
| 100                | 1        | 0          | 16      | 0       | 75505           |
| 500                | 9        | 11         | 97      | 0       | 52192           |
| 1000               | 19       | 24         | 187     | 0       | 50040           |
| 2000               | 39       | 54         | 409     | 0       | 42551           |
| 5000               | 96       | 128        | 1148    | 0       | 46367           |

- teleport/socket

| client concurrency | mean(ms) | median(ms) | max(ms) | min(ms) | throughput(TPS) |
| ------------------ | -------- | ---------- | ------- | ------- | --------------- |
| 100                | 0        | 0          | 14      | 0       | 225682          |
| 500                | 2        | 1          | 24      | 0       | 212630          |
| 1000               | 4        | 3          | 51      | 0       | 180733          |
| 2000               | 8        | 6          | 64      | 0       | 183351          |
| 5000               | 21       | 18         | 651     | 0       | 133886          |

**[test code](https://github.com/henrylee2cn/rpc-benchmark/tree/v4/teleport)**

- Profile torch of teleport/socket

![tp_socket_profile_torch](https://github.com/henrylee2cn/teleport/raw/v4/doc/tp_socket_profile_torch.png)

**[svg file](https://github.com/henrylee2cn/teleport/raw/v4/doc/tp_socket_profile_torch.svg)**

- Heap torch of teleport/socket

![tp_socket_heap_torch](https://github.com/henrylee2cn/teleport/raw/v4/doc/tp_socket_heap_torch.png)

**[svg file](https://github.com/henrylee2cn/teleport/raw/v4/doc/tp_socket_heap_torch.svg)**

## Version

| version | status  | branch                                   |
| ------- | ------- | ---------------------------------------- |
| v4      | pre-release | [v4](https://github.com/henrylee2cn/teleport/tree/v4) |
| v3      | release | [master(or v3)](https://github.com/henrylee2cn/teleport/tree/master) |
| v2      | release | [v2](https://github.com/henrylee2cn/teleport/tree/v2) |
| v1      | release | [v1](https://github.com/henrylee2cn/teleport/tree/v1) |


## Install

```sh
go get -u -f github.com/henrylee2cn/teleport
```

## Feature

- Server and client are peer-to-peer, have the same API method
- Support custom communication protocol
- Support set the size of socket I/O buffer
- Packet contains both `Header` and `Body` two parts
- Packet `Header` contains metadata in the same format as HTTP header
- Support for customizing `Body` coding types separately, e.g `JSON` `Protobuf` `string`
- Support push, call, reply and other means of communication
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
- Provide an operating interface to control the connection file descriptor

## Example

### server.go

```go
package main

import (
    "fmt"
    "time"

    tp "github.com/henrylee2cn/teleport"
)

func main() {
    srv := tp.NewPeer(tp.PeerConfig{
        CountTime:  true,
        ListenPort: 9090,
    })
    srv.RouteCall(new(math))
    srv.ListenAndServe()
}

type math struct {
    tp.CallCtx
}

func (m *math) Add(arg *[]int) (int, *tp.Rerror) {
    if m.Query().Get("push_status") == "yes" {
        m.Session().Push(
            "/push/status",
            fmt.Sprintf("%d numbers are being added...", len(*arg)),
        )
        time.Sleep(time.Millisecond * 10)
    }
    var r int
    for _, a := range *arg {
        r += a
    }
    return r, nil
}
```

### client.go

```go
package main

import (
    tp "github.com/henrylee2cn/teleport"
)

func main() {
    tp.SetLoggerLevel("ERROR")
    cli := tp.NewPeer(tp.PeerConfig{})
    defer cli.Close()
    cli.RoutePush(new(push))
    sess, err := cli.Dial(":9090")
    if err != nil {
        tp.Fatalf("%v", err)
    }

    var result int
    rerr := sess.Call("/math/add?push_status=yes",
        []int{1, 2, 3, 4, 5},
        &result,
    ).Rerror()

    if rerr != nil {
        tp.Fatalf("%v", rerr)
    }
    tp.Printf("result: %d", result)
}

type push struct {
    tp.PushCtx
}

func (p *push) Status(arg *string) *tp.Rerror {
    tp.Printf("server status: %s", *arg)
    return nil
}
```

[More Examples](https://github.com/henrylee2cn/teleport/tree/v4/examples)

## Design

### Keywords

- **Peer:** A communication instance may be a server or a client
- **Socket:** Base on the net.Conn package, add custom package protocol, transfer pipelines and other functions
- **Packet:** The corresponding structure of the data package content element
- **Proto:** The protocol interface of packet pack/unpack 
- **Codec:** Serialization interface for `Packet.Body`
- **XferPipe:** Packet bytes encoding pipeline, such as compression, encryption, calibration and so on
- **XferFilter:** A interface to handle packet data before transfer
- **Plugin:** Plugins that cover all aspects of communication
- **Session:** A connection session, with push, call, reply, close and other methods of operation
- **Context:** Handle the received or send packets
- **Call-Launch:** Call data from the peer
- **Call-Handle:** Handle and reply to the calling of peer
- **Push-Launch:** Push data to the peer
- **Push-Handle:** Handle the pushing of peer
- **Router:** Router that route the response handler by request information(such as a URI)

### Packet

The contents of every one packet:

```go
// in .../teleport/socket package

// Packet a socket data packet.
type Packet struct {
    // Has unexported fields.
}

func GetPacket(settings ...PacketSetting) *Packet
func NewPacket(settings ...PacketSetting) *Packet
func (p *Packet) Body() interface{}
func (p *Packet) BodyCodec() byte
func (p *Packet) Context() context.Context
func (p *Packet) MarshalBody() ([]byte, error)
func (p *Packet) Meta() *utils.Args
func (p *Packet) Ptype() byte
func (p *Packet) Reset(settings ...PacketSetting)
func (p *Packet) Seq() string
func (p *Packet) SetBody(body interface{})
func (p *Packet) SetBodyCodec(bodyCodec byte)
func (p *Packet) SetNewBody(newBodyFunc NewBodyFunc)
func (p *Packet) SetPtype(ptype byte)
func (p *Packet) SetSeq(seq string)
func (p *Packet) SetSize(size uint32) error
func (p *Packet) SetUri(uri string)
func (p *Packet) SetUriObject(uriObject *url.URL)
func (p *Packet) Size() uint32
func (p *Packet) String() string
func (p *Packet) UnmarshalBody(bodyBytes []byte) error
func (p *Packet) Uri() string
func (p *Packet) UriObject() *url.URL
func (p *Packet) XferPipe() *xfer.XferPipe

// NewBodyFunc creates a new body by header.
type NewBodyFunc func(Header) interface{}
```

### Codec

The body's codec set.

```go
type Codec interface {
    // Id returns codec id.
    Id() byte
    // Name returns codec name.
    Name() string
    // Marshal returns the encoding of v.
    Marshal(v interface{}) ([]byte, error)
    // Unmarshal parses the encoded data and stores the result
    // in the value pointed to by v.
    Unmarshal(data []byte, v interface{}) error
}
```

### XferPipe

Transfer filter pipe, handles byte stream of packet when transfer.

```go
// XferFilter handles byte stream of packet when transfer.
type XferFilter interface {
    // Id returns transfer filter id.
    Id() byte
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
// Note: the length can not be bigger than 255!
type XferPipe struct {
    // Has unexported fields.
}
func NewXferPipe() *XferPipe
func (x *XferPipe) Append(filterId ...byte) error
func (x *XferPipe) AppendFrom(src *XferPipe)
func (x *XferPipe) Ids() []byte
func (x *XferPipe) Len() int
func (x *XferPipe) Names() []string
func (x *XferPipe) OnPack(data []byte) ([]byte, error)
func (x *XferPipe) OnUnpack(data []byte) ([]byte, error)
func (x *XferPipe) Range(callback func(idx int, filter XferFilter) bool)
func (x *XferPipe) Reset()
```

### Plugin

Plug-ins during runtime.

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

### Protocol

You can customize your own communication protocol by implementing the interface:

```go
type (
    // Proto pack/unpack protocol scheme of socket packet.
    Proto interface {
        // Version returns the protocol's id and name.
        Version() (byte, string)
        // Pack writes the Packet into the connection.
        // Note: Make sure to write only once or there will be package contamination!
        Pack(*Packet) error
        // Unpack reads bytes from the connection to the Packet.
        // Note: Concurrent unsafe!
        Unpack(*Packet) error
    }
    ProtoFunc func(io.ReadWriter) Proto
)
```

Next, you can specify the communication protocol in the following ways:

```go
func SetDefaultProtoFunc(socket.ProtoFunc)
type Peer interface {
    ...
    ServeConn(conn net.Conn, protoFunc ...socket.ProtoFunc) Session
    DialContext(ctx context.Context, addr string, protoFunc ...socket.ProtoFunc) (Session, *Rerror)
    Dial(addr string, protoFunc ...socket.ProtoFunc) (Session, *Rerror)
    Listen(protoFunc ...socket.ProtoFunc) error
    ...
}
```

Default protocol `RawProto`(Big Endian):

```sh
{4 bytes packet length}
{1 byte protocol version}
{1 byte transfer pipe length}
{transfer pipe IDs}
# The following is handled data by transfer pipe
{4 bytes sequence length}
{sequence}
{1 byte packet type} // e.g. CALL:1; REPLY:2; PUSH:3
{4 bytes URI length}
{URI}
{4 bytes metadata length}
{metadata(urlencoded)}
{1 byte body codec id}
{body}
```


## Usage

### Peer(server or client) Demo

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

### Call-Controller-Struct API template

```go
type Aaa struct {
    tp.CallCtx
}
func (x *Aaa) XxZz(arg *<T>) (<T>, *tp.Rerror) {
    ...
    return r, nil
}
```

- register it to root router:

```go
// register the call route: /aaa/xx_zz
peer.RouteCall(new(Aaa))

// or register the call route: /xx_zz
peer.RouteCallFunc((*Aaa).XxZz)
```

### Call-Handler-Function API template

```go
func XxZz(ctx tp.CallCtx, arg *<T>) (<T>, *tp.Rerror) {
    ...
    return r, nil
}
```

- register it to root router:

```go
// register the call route: /xx_zz
peer.RouteCallFunc(XxZz)
```

### Push-Controller-Struct API template

```go
type Bbb struct {
    tp.PushCtx
}
func (b *Bbb) YyZz(arg *<T>) *tp.Rerror {
    ...
    return nil
}
```

- register it to root router:

```go
// register the push route: /bbb/yy_zz
peer.RoutePush(new(Bbb))

// or register the push route: /yy_zz
peer.RoutePushFunc((*Bbb).YyZz)
```

### Push-Handler-Function API template

```go
// YyZz register the route: /yy_zz
func YyZz(ctx tp.PushCtx, arg *<T>) *tp.Rerror {
    ...
    return nil
}
```

- register it to root router:

```go
// register the push route: /yy_zz
peer.RoutePushFunc(YyZz)
```

### Unknown-Call-Handler-Function API template

```go
func XxxUnknownCall (ctx tp.UnknownCallCtx) (interface{}, *tp.Rerror) {
    ...
    return r, nil
}
```

- register it to root router:

```go
// register the unknown call route: /*
peer.SetUnknownCall(XxxUnknownCall)
```

### Unknown-Push-Handler-Function API template

```go
func XxxUnknownPush(ctx tp.UnknownPushCtx) *tp.Rerror {
    ...
    return nil
}
```

- register it to root router:

```go
// register the unknown push route: /*
peer.SetUnknownPush(XxxUnknownPush)
```

### The mapping rule of struct(func) name to URI path:

- `AaBb` -> `/aa_bb`
- `Aa_Bb` -> `/aa/bb`
- `aa_bb` -> `/aa/bb`
- `Aa__Bb` -> `/aa_bb`
- `aa__bb` -> `/aa_bb`
- `ABC_XYZ` -> `/abc/xyz`
- `ABcXYz` -> `/abc_xyz`
- `ABC__XYZ` -> `/abc_xyz`

### Plugin Demo

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

func (i *ignoreCase) PostReadCallHeader(ctx tp.ReadCtx) *tp.Rerror {
    // Dynamic transformation path is lowercase
    ctx.UriObject().Path = strings.ToLower(ctx.UriObject().Path)
    return nil
}

func (i *ignoreCase) PostReadPushHeader(ctx tp.ReadCtx) *tp.Rerror {
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
    Network            string        `yaml:"network"              ini:"network"              comment:"Network; tcp, tcp4, tcp6, unix or unixpacket"`
    LocalIP            string        `yaml:"local_ip"             ini:"local_ip"             comment:"Local IP"`
    ListenPort         uint16        `yaml:"listen_port"          ini:"listen_port"          comment:"Listen port; for server role"`
    DefaultDialTimeout time.Duration `yaml:"default_dial_timeout" ini:"default_dial_timeout" comment:"Default maximum duration for dialing; for client role; ns,µs,ms,s,m,h"`
    RedialTimes        int32         `yaml:"redial_times"         ini:"redial_times"         comment:"The maximum times of attempts to redial, after the connection has been unexpectedly broken; for client role"`
    DefaultBodyCodec   string        `yaml:"default_body_codec"   ini:"default_body_codec"   comment:"Default body codec type id"`
    DefaultSessionAge  time.Duration `yaml:"default_session_age"  ini:"default_session_age"  comment:"Default session max age, if less than or equal to 0, no time limit; ns,µs,ms,s,m,h"`
    DefaultContextAge  time.Duration `yaml:"default_context_age"  ini:"default_context_age"  comment:"Default CALL or PUSH context max age, if less than or equal to 0, no time limit; ns,µs,ms,s,m,h"`
    SlowCometDuration  time.Duration `yaml:"slow_comet_duration"  ini:"slow_comet_duration"  comment:"Slow operation alarm threshold; ns,µs,ms,s ..."`
    PrintDetail        bool          `yaml:"print_detail"         ini:"print_detail"         comment:"Is print body and metadata or not"`
    CountTime          bool          `yaml:"count_time"           ini:"count_time"           comment:"Is count cost time or not"`
}
```

### Optimize

- SetPacketSizeLimit sets max packet size.
  If maxSize<=0, set it to max uint32.

    ```go
    func SetPacketSizeLimit(maxPacketSize uint32)
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
  packet transmission in hopes of sending fewer packets (Nagle's
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
| [json](https://github.com/henrylee2cn/teleport/blob/v4/codec/json_codec.go) | `import "github.com/henrylee2cn/teleport/codec"` | JSON codec(teleport own)     |
| [protobuf](https://github.com/henrylee2cn/teleport/blob/v4/codec/protobuf_codec.go) | `import "github.com/henrylee2cn/teleport/codec"` | Protobuf codec(teleport own) |
| [plain](https://github.com/henrylee2cn/teleport/blob/v4/codec/plain_codec.go) | `import "github.com/henrylee2cn/teleport/codec"` | Plain text codec(teleport own)   |
| [form](https://github.com/henrylee2cn/teleport/blob/v4/codec/form_codec.go) | `import "github.com/henrylee2cn/teleport/codec"` | Form(url encode) codec(teleport own)   |

### Plugin

| package                                  | import                                   | description                              |
| ---------------------------------------- | ---------------------------------------- | ---------------------------------------- |
| [auth](https://github.com/henrylee2cn/teleport/tree/v4/plugin/auth) | `import "github.com/henrylee2cn/teleport/plugin/auth"` | A auth plugin for verifying peer at the first time |
| [binder](https://github.com/henrylee2cn/teleport/tree/v4/plugin/binder) | `import binder "github.com/henrylee2cn/teleport/plugin/binder"` | Parameter Binding Verification for Struct Handler |
| [heartbeat](https://github.com/henrylee2cn/teleport/tree/v4/plugin/heartbeat) | `import heartbeat "github.com/henrylee2cn/teleport/plugin/heartbeat"` | A generic timing heartbeat plugin        |
| [proxy](https://github.com/henrylee2cn/teleport/tree/v4/plugin/proxy) | `import "github.com/henrylee2cn/teleport/plugin/proxy"` | A proxy plugin for handling unknown calling or pushing |
[secure](https://github.com/henrylee2cn/teleport/tree/v4/plugin/secure)|`import secure "github.com/henrylee2cn/teleport/plugin/secure"`|Encrypting/decrypting the packet body

### Protocol

| package                                  | import                                   | description                              |
| ---------------------------------------- | ---------------------------------------- | ---------------------------------------- |
| [rawproto](https://github.com/henrylee2cn/teleport/tree/v4/proto/rawproto) | `import "github.com/henrylee2cn/teleport/proto/rawproto` | A fast socket communication protocol(teleport default protocol) |
| [jsonproto](https://github.com/henrylee2cn/teleport/tree/v4/proto/jsonproto) | `import "github.com/henrylee2cn/teleport/proto/jsonproto"` | A JSON socket communication protocol     |
| [pbproto](https://github.com/henrylee2cn/teleport/tree/v4/proto/pbproto) | `import "github.com/henrylee2cn/teleport/proto/pbproto"` | A Protobuf socket communication protocol     |

### Transfer-Filter

| package                                  | import                                   | description                              |
| ---------------------------------------- | ---------------------------------------- | ---------------------------------------- |
| [gzip](https://github.com/henrylee2cn/teleport/tree/v4/xfer/gzip) | `import "github.com/henrylee2cn/teleport/xfer/gzip"` | Gzip(teleport own)                       |
| [md5](https://github.com/henrylee2cn/teleport/tree/v4/xfer/md5) | `import "github.com/henrylee2cn/teleport/xfer/md5"` | Provides a integrity check transfer filter |

### Mixer

| package                                  | import                                   | description                              |
| ---------------------------------------- | ---------------------------------------- | ---------------------------------------- |
| [clientsession](https://github.com/henrylee2cn/teleport/tree/v4/mixer/clientsession) | `import "github.com/henrylee2cn/teleport/mixer/clientsession"` | Client session with a high efficient and load balanced connection pool |
| [websocket](https://github.com/henrylee2cn/teleport/tree/v4/mixer/websocket) | `import "github.com/henrylee2cn/teleport/mixer/websocket"` | Makes the Teleport framework compatible with websocket protocol as specified in RFC 6455 |
| [html](https://github.com/xiaoenai/tp-micro/tree/master/helper/mod-html) | `html "github.com/xiaoenai/tp-micro/helper/mod-html"` | HTML render for http client |

## Projects based on Teleport

| project                                  | description                              |
| ---------------------------------------- | ---------------------------------------- |
| [TP-Micro](https://github.com/xiaoenai/tp-micro) | TP-Micro is a simple, powerful micro service framework based on Teleport |
| [Pholcus](https://github.com/henrylee2cn/pholcus) | Pholcus is a distributed, high concurrency and powerful web crawler software |

## Business Users

[![深圳市梦之舵信息技术有限公司](https://statics.xiaoenai.com/v4/img/logo_zh.png)](http://www.xiaoenai.com)
&nbsp;&nbsp;
[![北京风行在线技术有限公司](http://static.funshion.com/open/static/img/logo.gif)](http://www.fun.tv)
&nbsp;&nbsp;
[![北京可即时代网络公司](http://simg.ktvms.com/picture/logo.png)](http://www.kejishidai.cn)

## License

Teleport is under Apache v2 License. See the [LICENSE](https://github.com/henrylee2cn/teleport/raw/v4/LICENSE) file for the full license text
