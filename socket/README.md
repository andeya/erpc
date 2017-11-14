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
- Support custom communication protocol
- Each socket is assigned an id
- Provides `Socket` hub, `Socket` pool and `*Packet` stack

## Packet

The contents of every one packet:

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
	// packet size
	Size int64 `json:"size"`
}
```

Among the contents of the header:

```go
type Header struct {
	// Packet id
	Id string
	// Service type
	Type int32
	// Service URI
	Uri string
	// Body gzip level [-2,9]
	Gzip int32
	// As reply, it indicates the service status code
	StatusCode int32
	// As reply, it indicates the service status text
	Status string
}
```

## Protocol

The default socket communication protocol:

```
HeaderLength | HeaderCodecId | Header | BodyLength | BodyCodecId | Body
```

**Notes:**

- `HeaderLength`: uint32, 4 bytes, big endian
- `HeaderCodecId`: uint8, 1 byte
- `Header`: header bytes
- `BodyLength`: uint32, 4 bytes, big endian
	* may be 0, meaning that the `Body` is empty and does not indicate the `BodyCodecId`
	* may be 1, meaning that the `Body` is empty but indicates the `BodyCodecId`
- `BodyCodecId`: uint8, 1 byte
- `Body`: body bytes

You can customize your own communication protocol by implementing the interface:

```go
// Protocol socket communication protocol
type Protocol interface {
	// WritePacket writes header and body to the connection.
	WritePacket(
		packet *Packet,
		destWriter *utils.BufioWriter,
		tmpCodecWriterGetter func(codecName string) (*TmpCodecWriter, error),
		isActiveClosed func() bool,
	) error

	// ReadPacket reads header and body from the connection.
	ReadPacket(
		packet *Packet,
		bodyAdapter func() interface{},
		srcReader *utils.BufioReader,
		codecReaderGetter func(codecId byte) (*CodecReader, error),
		isActiveClosed func() bool,
		checkReadLimit func(int64) error,
	) error
}
```

Next, you can specify the communication protocol in the following ways:

```go
func SetDefaultProtocol(Protocol)
func GetSocket(net.Conn, ...Protocol) Socket
func NewSocket(net.Conn, ...Protocol) Socket
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