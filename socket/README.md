# Socket

A concise, powerful and high-performance TCP connection socket.

## Feature

- The server and client are peer-to-peer interfaces
- With I/O buffer
- Packet contains both Header and Body
- Supports custom encoding types, e.g `JSON` `Protobuf`
- Header and Body can use different coding types
- Body supports gzip compression
- Header contains the status code and its description text
- Each socket is assigned an id
- Provides `Socket` hub, `Socket` pool and `*Packet` stack

## Packet

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

## Header

```go
type Header struct {
	// Packet id
	Id string
	// Service type
	Type int32
	// Service URI
	Uri string
	// Body gizp compression level
	Gzip int32
	// As reply, it indicates the service status code
	StatusCode int32
	// As reply, it indicates the service status text
	Status string
}
```

## Demo

### server.go

```go
package main

import (
	"log"
	"net"
	"time"

	"github.com/henrylee2cn/teleport/socket"
)

func main() {
	lis, err := net.Listen("tcp", "0.0.0.0:8000")
	if err != nil {
		log.Fatalf("[SVR] listen err: %v", err)
	}
	log.Printf("listen tcp 0.0.0.0:8000")
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Fatalf("[SVR] accept err: %v", err)
		}
		go func(s socket.Socket) {
			log.Printf("accept %s", s.Id())
			defer s.Close()
			for {
				// read request
				var packet = socket.GetPacket(func(_ *socket.Header) interface{} {
					return new(map[string]string)
				})
				err = s.ReadPacket(packet)
				if err != nil {
					log.Printf("[SVR] read request err: %v", err)
					return
				} else {
					log.Printf("[SVR] read request: %v", packet)
				}

				// write response
				packet.HeaderCodec = "json"
				packet.BodyCodec = "json"
				packet.Header.StatusCode = 200
				packet.Header.Status = "ok"
				packet.Body = time.Now()
				err = s.WritePacket(packet)
				if err != nil {
					log.Printf("[SVR] write response err: %v", err)
				}
				log.Printf("[SVR] write response: %v", packet)
				socket.PutPacket(packet)
			}
		}(socket.GetSocket(conn))
	}
}
```

### client.go

```go
package main

import (
	"log"
	"net"

	"github.com/henrylee2cn/teleport/socket"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:8000")
	if err != nil {
		log.Fatalf("[CLI] dial err: %v", err)
	}
	s := socket.GetSocket(conn)
	defer s.Close()
	var packet = socket.GetPacket(nil)
	defer socket.PutPacket(packet)
	for i := 0; i < 10; i++ {
		// write request
		packet.Reset(nil)
		packet.HeaderCodec = "json"
		packet.BodyCodec = "json"
		packet.Header.Seq = 1
		packet.Header.Uri = "/a/b"
		packet.Header.Gzip = 5
		packet.Body = map[string]string{"a": "A"}
		err = s.WritePacket(packet)
		if err != nil {
			log.Printf("[CLI] write request err: %v", err)
			continue
		}
		log.Printf("[CLI] write request: %v", packet)

		// read response
		packet.Reset(func(_ *socket.Header) interface{} {
			return new(string)
		})
		err = s.ReadPacket(packet)
		if err != nil {
			log.Printf("[CLI] read response err: %v", err)
		} else {
			log.Printf("[CLI] read response: %v", packet)
		}
	}
}
```