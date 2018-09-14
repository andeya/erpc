# Socket

A concise, powerful and high-performance connection socket.

## Feature

- The server and client are peer-to-peer interfaces
- Support set the size of socket I/O buffer
- Support custom communication protocol
- Support custom transfer filter pipe (Such as gzip, encrypt, verify...)
- Message contains both Header and Body
- Supports custom encoding types, e.g `JSON` `Protobuf`
- Header contains the status code and its description text
- Each socket is assigned an id
- Provides `Socket Hub`, `Socket` pool and `*Message` stack
- Support setting the size of the reading message (if exceed disconnect it)
- Provide an operating interface to control the connection file descriptor


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
	socket.SetMessageSizeLimit(512)
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
				var message = socket.GetMessage(socket.WithNewBody(
					func(header socket.Header) interface{} {
						*pbTest = pb.PbTest{}
						return pbTest
					}),
				)
				err = s.ReadMessage(message)
				if err != nil {
					log.Printf("[SVR] read request err: %v", err)
					return
				} else {
					// log.Printf("[SVR] read request: %v", message)
				}

				// write response
				pbTest.A = pbTest.A + pbTest.B
				pbTest.B = pbTest.A - pbTest.B*2
				message.SetBody(pbTest)

				err = s.WriteMessage(message)
				if err != nil {
					log.Printf("[SVR] write response err: %v", err)
				} else {
					// log.Printf("[SVR] write response: %v", message)
				}
				socket.PutMessage(message)
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
	var message = socket.GetMessage()
	defer socket.PutMessage(message)
	for i := uint64(0); i < 1; i++ {
		// write request
		message.Reset()
		message.SetMtype(0)
		message.SetBodyCodec(codec.ID_JSON)
		message.SetSeq(i)
		message.SetUri("/a/b")
		message.SetBody(&pb.PbTest{A: 10, B: 2})
		err = s.WriteMessage(message)
		if err != nil {
			log.Printf("[CLI] write request err: %v", err)
			continue
		}
		log.Printf("[CLI] write request: %v", message)

		// read response
		message.Reset(socket.WithNewBody(
			func(header socket.Header) interface{} {
				return new(pb.PbTest)
			}),
		)
		err = s.ReadMessage(message)
		if err != nil {
			log.Printf("[CLI] read response err: %v", err)
		} else {
			log.Printf("[CLI] read response: %v", message)
		}
	}
}
```

[More Examples](https://github.com/henrylee2cn/teleport/tree/v4/socket/example)

## Keyworks

- **Message:** The corresponding structure of the data package
- **Proto:** The protocol interface of message pack/unpack 
- **Codec:** Serialization interface for `Message.Body`
- **XferPipe:** A series of pipelines to handle message data before transfer
- **XferFilter:** A interface to handle message data before transfer


## Message

The contents of every one message:

```go
// in .../teleport/socket package
type (
	type Message struct {
		// Has unexported fields.
	}
	    Message a socket data message.
	
	func GetMessage(settings ...MessageSetting) *Message
	func NewMessage(settings ...MessageSetting) *Message
	func (m *Message) Body() interface{}
	func (m *Message) BodyCodec() byte
	func (m *Message) Context() context.Context
	func (m *Message) MarshalBody() ([]byte, error)
	func (m *Message) Meta() *utils.Args
	func (m *Message) Mtype() byte
	func (m *Message) Reset(settings ...MessageSetting)
	func (m *Message) Seq() string
	func (m *Message) SetBody(body interface{})
	func (m *Message) SetBodyCodec(bodyCodec byte)
	func (m *Message) SetNewBody(newBodyFunc NewBodyFunc)
	func (m *Message) SetMtype(mtype byte)
	func (m *Message) SetSeq(seq string)
	func (m *Message) SetSize(size uint32) error
	func (m *Message) SetUri(uri string)
	func (m *Message) SetUriObject(uriObject *url.URL)
	func (m *Message) Size() uint32
	func (m *Message) String() string
	func (m *Message) UnmarshalBody(bodyBytes []byte) error
	func (m *Message) Uri() string
	func (m *Message) UriObject() *url.URL
	func (m *Message) XferPipe() *xfer.XferPipe

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
	// XferFilter handles byte stream of message when transfer.
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
	// Proto pack/unpack protocol scheme of socket message.
	Proto interface {
		// Version returns the protocol's id and name.
		Version() (byte, string)
		// Pack writes the Message into the connection.
		// Note: Make sure to write only once or there will be package contamination!
		Pack(*Message) error
		// Unpack reads bytes from the connection to the Message.
		// Note: Concurrent unsafe!
		Unpack(*Message) error
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

Default protocol `RawProto`(Big Endian):

```sh
{4 bytes message length}
{1 byte protocol version}
{1 byte transfer pipe length}
{transfer pipe IDs}
# The following is handled data by transfer pipe
{2 bytes sequence length}
{sequence}
{1 byte message type} # e.g. CALL:1; REPLY:2; PUSH:3
{2 bytes URI length}
{URI}
{2 bytes metadata length}
{metadata(urlencoded)}
{1 byte body codec id}
{body}
```

## Optimize

- SetMessageSizeLimit sets max message size.
If maxSize<=0, set it to max uint32.

	```go
	func SetMessageSizeLimit(maxMessageSize uint32)
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
message transmission in hopes of sending fewer messages (Nagle's
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
