# teleport [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/teleport?style=flat-square)](http://goreportcard.com/report/henrylee2cn/teleport) [![github issues](https://img.shields.io/github/issues/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aopen+is%3Aissue) [![github closed issues](https://img.shields.io/github/issues-closed-raw/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aissue+is%3Aclosed) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/henrylee2cn/teleport) [![view examples](https://img.shields.io/badge/learn%20by-examples-00BCD4.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/tree/master/samples)
<!--  [![GitHub release](https://img.shields.io/github/release/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/releases) -->

Teleport is a versatile, high-performance and flexible TCP socket framework.

It can be used for rpc, micro services, peer-peer, push services, game services and so on.

![teleport_server](https://github.com/henrylee2cn/teleport/raw/master/doc/teleport_server.png)
<br>
<center><i>AB Testing 1: [Mac 4CPU 8GB] [single-process single-conn] teleport: QPS 35448</i></center>

![teleport_frame_client_ab_test](https://github.com/henrylee2cn/teleport/raw/master/doc/frame_client_ab.png)
<br>
<center><i>AB Testing 2: [Mac 4CPU 8GB] [single-process single-conn] teleport/socket: QPS 54040</i></center>

![teleport_socket_client_ab_test](https://github.com/henrylee2cn/teleport/raw/master/doc/socket_client_ab.png)

## 1. Version

version | status | branch
--------|--------|--------
v2      | beta | [master](https://github.com/henrylee2cn/teleport/tree/master)
v1      | release | [v1](https://github.com/henrylee2cn/teleport/tree/v1)


## 2. Install

```sh
go get -u github.com/henrylee2cn/teleport
```

## 3. Feature

- Server and client are peer-to-peer, have the same API method
- Packet contains both Header and Body two parts
- Support for customizing head and body coding types separately, e.g `JSON` `Protobuf`
- Body supports gzip compression
- Header contains the status code and its description text
- Support push, pull, reply and other means of communication
- Support plug-in mechanism, can customize authentication, heartbeat, micro service registration center, statistics, etc.
- Whether server or client, the peer support reboot and shutdown gracefully
- Support reverse proxy
- Detailed log information, support print input and output details
- Supports setting slow operation alarm threshold
- With a connection I/O buffer
- Use I/O multiplexing technology

## 4. Architecture

### 4.1 Keywords

- **Peer:** A communication instance may be a client or a client
- **Session:** A connection session, with push, pull, reply, close and other methods of operation
- **readHandleCtx:** Handle the received packets
- **Pull-Launch:** Pull data from the peer
- **Pull-Handle:** Handle and reply to the pull of peer
- **Push-Launch:** Push data to the peer
- **Push-Handle:** Handle the push of peer
- **Router:** Register handlers


### 4.2 Execution level

```
Peer -> Session -> Socket -> Conn -> ApiContext
```


### 4.3 Packet

```
HeaderLength | HeaderCodecId | Header | BodyLength | BodyCodecId | Body
```

**Notes:**

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

### 4.4 Header

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

## 5. Usage

- Create a server or client peer

```go
var cfg = &teleport.PeerConfig{
	DefaultReadTimeout:  time.Minute * 3,
	DefaultWriteTimeout: time.Minute * 3,
	TlsCertFile:         "",
	TlsKeyFile:          "",
	SlowCometDuration:   time.Millisecond * 500,
	DefaultCodec:        "json",
	DefaultGzipLevel:    5,
	PrintBody:           true,
	DefaultDialTimeout:  time.Second * 10,
	ListenAddrs: []string{
		"0.0.0.0:9090",
	},
}


var peer = teleport.NewPeer(cfg)

// It can be used as a server
peer.Listen()

// It can also be used as a client at the same time
var sess, err = peer.Dial("127.0.0.1:8080", "peerid-client")
if err != nil {
	teleport.Panicf("%v", err)
}
```

- Define a controller and handler for pull request

```go
// Home controller
type Home struct {
	teleport.PullCtx
}

// Test handler
func (h *Home) Test(args *[2]int) (int, teleport.Xerror) {
	a := (*args)[0]
	b := (*args)[1]
	return a + b, nil
}
```

- Define a controller and handler for push request

```go
// Msg controller
type Msg struct {
	teleport.PushCtx
}

// Test handler
func (m *Msg) Test(args *map[string]interface{}) {
	teleport.Infof("receive push(%s):\nargs: %#v\nquery: %#v\n", m.Ip(), args, m.Query())
}
```

- Define a handler for unknown pull request

```go
func UnknownPullHandle(ctx teleport.UnknownPullCtx, body *[]byte) (interface{}, teleport.Xerror) {
	var v interface{}
	codecName, err := ctx.Unmarshal(*body, &v, true)
	if err != nil {
		return nil, teleport.NewXerror(0, err.Error())
	}
	teleport.Infof("receive unknown pull:\n codec: %s\n content: %#v", codecName, v)
	return "this is reply string for unknown pull", nil
}

```

- Define a handler for unknown push request

```go
func UnknownPushHandle(ctx teleport.UnknownPushCtx, body *[]byte) {
	var v interface{}
	codecName, err := ctx.Unmarshal(*body, &v, true)
	if err != nil {
		teleport.Errorf("%v", err)
	} else {
		teleport.Infof("receive unknown push:\n codec: %s\n content: %#v", codecName, v)
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

// PostReadHeader converts the alias of this service.
func (p *AliasPlugin) PostReadHeader(ctx teleport.ReadCtx) error {
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
	"time"

	"github.com/henrylee2cn/teleport"
)

func main() {
	go teleport.GraceSignal()
	teleport.SetShutdown(time.Second*20, nil, nil)
	var cfg = &teleport.PeerConfig{
		DefaultReadTimeout:       time.Minute * 3,
		DefaultWriteTimeout:      time.Minute * 3,
		TlsCertFile:       "",
		TlsKeyFile:        "",
		SlowCometDuration: time.Millisecond * 500,
		DefaultCodec:      "json",
		DefaultGzipLevel:  5,
		PrintBody:         true,
		ListenAddrs: []string{
			"0.0.0.0:9090",
			"0.0.0.0:9091",
		},
	}
	var peer = teleport.NewPeer(cfg)
	{
		group := peer.PullRouter.Group("group")
		group.Reg(new(Home))
	}
	peer.PullRouter.SetUnknown(UnknownPullHandle)
	peer.Listen()
}

// Home controller
type Home struct {
	teleport.PullCtx
}

// Test handler
func (h *Home) Test(args *map[string]interface{}) (map[string]interface{}, teleport.Xerror) {
	h.Session().Push("/push/test?tag=from home-test", map[string]interface{}{
		"your_id": h.Query().Get("peer_id"),
		"a":       1,
	})
	return map[string]interface{}{
		"your_args":   *args,
		"server_time": time.Now(),
	}, nil
}

func UnknownPullHandle(ctx teleport.UnknownPullCtx, body *[]byte) (interface{}, teleport.Xerror) {
	var v interface{}
	codecName, err := ctx.Unmarshal(*body, &v, true)
	if err != nil {
		return nil, teleport.NewXerror(0, err.Error())
	}
	teleport.Debugf("unmarshal body: codec: %s, content: %#v", codecName, v)
	return []string{"a", "aa", "aaa"}, nil
}
```

### client.go

```go
package main

import (
	"time"

	"github.com/henrylee2cn/teleport"
)

func main() {
	go teleport.GraceSignal()
	teleport.SetShutdown(time.Second*20, nil, nil)
	var cfg = &teleport.PeerConfig{
		DefaultReadTimeout:  time.Minute * 3,
		DefaultWriteTimeout: time.Minute * 3,
		TlsCertFile:         "",
		TlsKeyFile:          "",
		SlowCometDuration:   time.Millisecond * 500,
		DefaultCodec:        "json",
		DefaultGzipLevel:    5,
		PrintBody:           false,
	}

	var peer = teleport.NewPeer(cfg)
	peer.PushRouter.Reg(new(Push))

	{
		var sess, err = peer.Dial("127.0.0.1:9090", "simple_server:9090")
		if err != nil {
			teleport.Panicf("%v", err)
		}

		var reply interface{}
		var pullcmd = sess.Pull(
			"/group/home/test?peer_id=client9090",
			map[string]interface{}{"conn_port": 9090},
			&reply,
		)

		if pullcmd.Xerror != nil {
			teleport.Fatalf("pull error: %v", pullcmd.Xerror.Error())
		}
		teleport.Infof("9090reply: %#v", reply)
	}

	{
		var sess, err = peer.Dial("127.0.0.1:9091")
		if err != nil {
			teleport.Panicf("%v", err)
		}

		var reply interface{}
		var pullcmd = sess.Pull(
			"/group/home/test_unknown?peer_id=client9091",
			map[string]interface{}{"conn_port": 9091},
			&reply,
		)

		if pullcmd.Xerror != nil {
			teleport.Fatalf("pull error: %v", pullcmd.Xerror.Error())
		}
		teleport.Infof("9091reply test_unknown: %#v", reply)
	}
}

// Push controller
type Push struct {
	teleport.PushCtx
}

// Test handler
func (p *Push) Test(args *map[string]interface{}) {
	teleport.Infof("receive push(%s):\nargs: %#v\nquery: %#v\n", p.Ip(), args, p.Query())
}
```