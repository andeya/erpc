# teleport [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/teleport?style=flat-square)](http://goreportcard.com/report/henrylee2cn/teleport) [![github issues](https://img.shields.io/github/issues/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aopen+is%3Aissue) [![github closed issues](https://img.shields.io/github/issues-closed-raw/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/issues?q=is%3Aissue+is%3Aclosed) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/henrylee2cn/teleport) 
<!--  [![GitHub release](https://img.shields.io/github/release/henrylee2cn/teleport.svg?style=flat-square)](https://github.com/henrylee2cn/teleport/releases) -->

Teleport is a versatile, high-performance and flexible network communication package.

It can be used for RPC, micro services, peer-peer, push services, game services and so on.


## Version

version | status | branch
--------|--------|--------
v2      | beta | [master](https://github.com/henrylee2cn/teleport/tree/master)
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


![teleport_server](https://github.com/henrylee2cn/teleport/raw/master/doc/teleport_server.png)

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
		ReadTimeout:              time.Second * 10,
		WriteTimeout:             time.Second * 10,
		TlsCertFile:              "",
		TlsKeyFile:               "",
		SlowCometDuration:        time.Millisecond * 500,
		DefaultCodec:             "json",
		DefaultGzipLevel:         5,
		MaxGoroutinesAmount:      1024,
		MaxGoroutineIdleDuration: time.Second * 10,
		PrintBody:                true,
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
		ReadTimeout:              time.Second * 10,
		WriteTimeout:             time.Second * 10,
		TlsCertFile:              "",
		TlsKeyFile:               "",
		SlowCometDuration:        time.Millisecond * 500,
		DefaultCodec:             "json",
		DefaultGzipLevel:         5,
		MaxGoroutinesAmount:      1024,
		MaxGoroutineIdleDuration: time.Second * 10,
		PrintBody:                false,
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
			"/group/home/test?peer_id=client9091",
			map[string]interface{}{"conn_port": 9091},
			&reply,
		)

		if pullcmd.Xerror != nil {
			teleport.Fatalf("pull error: %v", pullcmd.Xerror.Error())
		}
		teleport.Infof("9091reply: %#v", reply)
	}
}

// Push controller
type Push struct {
	teleport.PushCtx
}

// Test handler
func (p *Push) Test(args *map[string]interface{}) {
	teleport.Infof("push-test(%s):\nargs: %#v\nquery: %#v\n", p.Ip(), args, p.Query())
}
```