# Socket

A concise, powerful and high-performance connection socket.

## Feature

- The server and client are peer-to-peer interfaces
- Support set the size of socket I/O buffer
- Support custom communication protocol
- Support custom transfer filter pipe (Such as gzip, encrypt, verify...)
- Packet contains both Header and Body
- Supports custom encoding types, e.g `JSON` `Protobuf`
- Header contains the status code and its description text
- Each socket is assigned an id
- Provides `Socket Hub`, `Socket` pool and `*Packet` stack
- Support setting the size of the reading packet (if exceed disconnect it)
- Provide an operating interface to control the connection file descriptor


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

## Example

### server.go

```go
package main

import (
	"log"
	"net"

	"github.com/henrylee2cn/teleport/socket"
	"github.com/henrylee2cn/teleport/socket/example/pb"
)

func main() {
	socket.SetPacketSizeLimit(512)
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
			var pbTest = new(pb.PbTest)
			for {
				// read request
				var packet = socket.GetPacket(socket.WithNewBody(
					func(header socket.Header) interface{} {
						*pbTest = pb.PbTest{}
						return pbTest
					}),
				)
				err = s.ReadPacket(packet)
				if err != nil {
					log.Printf("[SVR] read request err: %v", err)
					return
				} else {
					// log.Printf("[SVR] read request: %v", packet)
				}

				// write response
				pbTest.A = pbTest.A + pbTest.B
				pbTest.B = pbTest.A - pbTest.B*2
				packet.SetBody(pbTest)

				err = s.WritePacket(packet)
				if err != nil {
					log.Printf("[SVR] write response err: %v", err)
				} else {
					// log.Printf("[SVR] write response: %v", packet)
				}
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

	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/socket"

	"github.com/henrylee2cn/teleport/socket/example/pb"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:8000")
	if err != nil {
		log.Fatalf("[CLI] dial err: %v", err)
	}
	s := socket.GetSocket(conn)
	defer s.Close()
	var packet = socket.GetPacket()
	defer socket.PutPacket(packet)
	for i := uint64(0); i < 1; i++ {
		// write request
		packet.Reset()
		packet.SetPtype(0)
		packet.SetBodyCodec(codec.ID_JSON)
		packet.SetSeq(i)
		packet.SetUri("/a/b")
		packet.SetBody(&pb.PbTest{A: 10, B: 2})
		err = s.WritePacket(packet)
		if err != nil {
			log.Printf("[CLI] write request err: %v", err)
			continue
		}
		log.Printf("[CLI] write request: %v", packet)

		// read response
		packet.Reset(socket.WithNewBody(
			func(header socket.Header) interface{} {
				return new(pb.PbTest)
			}),
		)
		err = s.ReadPacket(packet)
		if err != nil {
			log.Printf("[CLI] read response err: %v", err)
		} else {
			log.Printf("[CLI] read response: %v", packet)
		}
	}
}
```

[More Examples](https://github.com/henrylee2cn/teleport/tree/master/socket/example)

## Keyworks

- **Packet:** The corresponding structure of the data package
- **Proto:** The protocol interface of packet pack/unpack 
- **Codec:** Serialization interface for `Packet.Body`
- **XferPipe:** A series of pipelines to handle packet data before transfer
- **XferFilter:** A interface to handle packet data before transfer


## Packet

The contents of every one packet:

```go
// in .../teleport/socket package
type (
	type Packet struct {
		// Has unexported fields.
	}
	    Packet a socket data packet.
	
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
	NewBodyFunc func(Header) interface{}
)

// in .../teleport/xfer package
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

## Protocol

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
func SetDefaultProtoFunc(ProtoFunc)
func GetSocket(net.Conn, ...ProtoFunc) Socket
func NewSocket(net.Conn, ...ProtoFunc) Socket
```

Default protocol `FastProto`(Big Endian):

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

## Optimize

- SetPacketSizeLimit sets max packet size.
If maxSize<=0, set it to max uint32.

	```go
	func SetPacketSizeLimit(maxPacketSize uint32)
	```

- SetKeepAlive sets whether the operating system should send
keepalive messages on the connection.

	```go
	func SetKeepAlive(keepalive bool)
	```

- SetKeepAlivePeriod sets period between keep alives.

	```go
	func SetKeepAlivePeriod(d time.Duration)
	```

- SetNoDelay controls whether the operating system should delay
packet transmission in hopes of sending fewer packets (Nagle's
algorithm).  The default is true (no delay), meaning that data is
sent as soon as possible after a Write.

	```go
	func SetNoDelay(_noDelay bool)
	```

- SetReadBuffer sets the size of the operating system's
receive buffer associated with the connection.

	```go
	func SetReadBuffer(bytes int)
	```

- SetWriteBuffer sets the size of the operating system's
transmit buffer associated with the connection.

	```go
	func SetWriteBuffer(bytes int)
	```
