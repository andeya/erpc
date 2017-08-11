# Socket

A concise, powerful and high-performance TCP connection socket.

## Feature

- The server and client are peer-to-peer interfaces
- With I/O buffer
- Packet contains both Header and Body
- Supports custom encoding types, e.g JSON
- Header and Body can use different coding types
- Body supports gzip compression
- Header contains the status code and its description text
- Each socket is assigned an id
- Provides a socket hub

## Packet

```
{Header-Length}{Header}{Body-Length}{Body}
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
	// Body encoding type
	Codec string
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
				var (
					header *socket.Header
					body   interface{}
				)
				n, err := s.ReadPacket(func(h *socket.Header) interface{} {
					header = h
					return &body
				})
				if err != nil {
					log.Printf("[SVR] read request err: %v", err)
					return
				} else {
					log.Printf("[SVR] read request len: %d, header:%#v, body: %#v", n, header, body)
				}

				// write response
				header.StatusCode = 1
				header.Status = "test error"
				now := time.Now()
				n, err = s.WritePacket(header, now)
				if err != nil {
					log.Printf("[SVR] write response err: %v", err)
				}
				log.Printf("[SVR] write response len: %d, header: %#v, body: %#v", n, header, now)
			}
		}(socket.Wrap(conn))
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
	s := socket.Wrap(conn)
	defer s.Close()
	for i := 0; i < 10; i++ {
		// write request
		header := &socket.Header{
			Id:    "1",
			Uri:   "/a/b",
			Codec: "json",
			Gzip:  5,
		}
		var body interface{} = "ABC"
		n, err := s.WritePacket(header, body)
		if err != nil {
			log.Printf("[CLI] write request err: %v", err)
			continue
		}
		log.Printf("[CLI] write request len: %d, header: %#v body: %#v", n, header, body)

		// read response
		n, err = s.ReadPacket(func(h *socket.Header) interface{} {
			header = h
			return &body
		})
		if err != nil {
			log.Printf("[CLI] read response err: %v", err)
		} else {
			log.Printf("[CLI] read response len: %d, header:%#v, body: %#v", n, header, body)
		}
	}
}
```