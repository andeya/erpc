# teleport [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/teleport?style=flat-square)](http://goreportcard.com/report/henrylee2cn/teleport) [![github issues](https://img.shields.io/github/issues/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aopen+is%3Aissue) [![github closed issues](https://img.shields.io/github/issues-closed-raw/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aissue+is%3Aclosed) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/henrylee2cn/teleport) 
<!--  [![GitHub release](https://img.shields.io/github/release/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/releases) -->

Teleport is a versatile, high-performance and flexible network communication package.

It can be used for RPC, micro services, peer-peer, push services, game services and so on.


## Version

version | status | branch
--------|--------|--------
v2      | developing | [master](https://github.com/henrylee2cn/teleport/tree/master)
v1      | release | [v1](https://github.com/henrylee2cn/teleport/tree/v1)


## Architecture

- Execution level

```
Peer -> Session -> Socket -> Conn -> ApiContext
```

## Socket

A concise, powerful and high-performance TCP connection socket.

### Feature

- The server and client are peer-to-peer interfaces
- With I/O buffer
- Packet contains both Header and Body
- Supports custom encoding types, e.g JSON
- Header and Body can use different coding types
- Body supports gzip compression
- Header contains the status code and its description text
- Each socket is assigned an id
- Provides `Socket` hub, `Socket` pool and `*Packet` stack

### Packet

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
	// HeaderCodec header codec string
	HeaderCodec string
	// BodyCodec body codec string
	BodyCodec string
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

### Header

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

### Demo

#### server.go


```go
package main

import (
	"time"

	"github.com/henrylee2cn/teleport"
)

func main() {
	var cfg = &teleport.Config{
		Id:                       "server-peer",
		ReadTimeout:              time.Second * 10,
		WriteTimeout:             time.Second * 10,
		TlsCertFile:              "",
		TlsKeyFile:               "",
		SlowCometDuration:        time.Millisecond * 500,
		DefaultCodec:             "json",
		DefaultGzipLevel:         5,
		MaxGoroutinesAmount:      1024,
		MaxGoroutineIdleDuration: time.Second * 10,
		ListenAddrs: []string{
			"0.0.0.0:9090",
			"0.0.0.0:9091",
		},
	}
	var peer = teleport.NewPeer(cfg)
	{
		group := peer.RequestRouter.Group("group")
		group.Reg(new(Home))
	}
	peer.Listen()
}

// Home controller
type Home struct {
	teleport.RequestCtx
}

// Test handler
func (h *Home) Test(args *string) (string, teleport.Xerror) {
	teleport.Infof("query: %#v", h.Query())
	return "home-test response:" + *args, nil
}
```

#### client.go

```go
package main

import (
	"time"

	"github.com/henrylee2cn/teleport"
)

func main() {
	var cfg = &teleport.Config{
		Id:                       "client-peer",
		ReadTimeout:              time.Second * 10,
		WriteTimeout:             time.Second * 10,
		TlsCertFile:              "",
		TlsKeyFile:               "",
		SlowCometDuration:        time.Millisecond * 500,
		DefaultCodec:             "json",
		DefaultGzipLevel:         5,
		MaxGoroutinesAmount:      1024,
		MaxGoroutineIdleDuration: time.Second * 10,
	}

	var peer = teleport.NewPeer(cfg)

	{
		var sess, err = peer.Dial("127.0.0.1:9090")
		if err != nil {
			teleport.Panicf("%v", err)
		}
		var reply string
		var pullcmd = sess.Pull("/group/home/test?port=9090", "test_args_9090", &reply)
		if pullcmd.Xerror != nil {
			teleport.Fatalf("pull error: %v", pullcmd.Xerror.Error())
		}
		teleport.Infof("9090reply: %s", reply)
	}

	{
		var sess, err = peer.Dial("127.0.0.1:9091")
		if err != nil {
			teleport.Panicf("%v", err)
		}
		var reply string
		var pullcmd = sess.Pull("/group/home/test?port=9091", "test_args_9091", &reply)
		if pullcmd.Xerror != nil {
			teleport.Fatalf("pull error: %v", pullcmd.Xerror.Error())
		}
		teleport.Infof("9091reply: %s", reply)
	}
}
```