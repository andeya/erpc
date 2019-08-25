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
- Provides `Socket Hub`, `Socket` pool and `Message` stack
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

**[test code](https://github.com/henrylee2cn/rpc-benchmark/tree/master/teleport)**

- Profile torch of teleport/socket

![tp_socket_profile_torch](https://github.com/henrylee2cn/teleport/raw/master/doc/tp_socket_profile_torch.png)

**[svg file](https://github.com/henrylee2cn/teleport/raw/master/doc/tp_socket_profile_torch.svg)**

- Heap torch of teleport/socket

![tp_socket_heap_torch](https://github.com/henrylee2cn/teleport/raw/master/doc/tp_socket_heap_torch.png)

**[svg file](https://github.com/henrylee2cn/teleport/raw/master/doc/tp_socket_heap_torch.svg)**

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

//go:generate go build $GOFILE

func main() {
    socket.SetNoDelay(false)
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
            log.Printf("accept %s", s.ID())
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
                }

                // write response
                pbTest.A = pbTest.A + pbTest.B
                pbTest.B = pbTest.A - pbTest.B*2
                message.SetBody(pbTest)

                err = s.WriteMessage(message)
                if err != nil {
                    log.Printf("[SVR] write response err: %v", err)
                } else {
                    log.Printf("[SVR] write response: %v", message)
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

//go:generate go build $GOFILE

func main() {
    conn, err := net.Dial("tcp", "127.0.0.1:8000")
    if err != nil {
        log.Fatalf("[CLI] dial err: %v", err)
    }
    s := socket.GetSocket(conn)
    defer s.Close()
    var message = socket.GetMessage()
    defer socket.PutMessage(message)
    for i := int32(0); i < 1; i++ {
        // write request
        message.Reset()
        message.SetMtype(0)
        message.SetBodyCodec(codec.ID_JSON)
        message.SetSeq(i)
        message.SetServiceMethod("/a/b")
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

[More Examples](https://github.com/henrylee2cn/teleport/tree/master/socket/example)

## Keyworks

- *Message:** The corresponding structure of the data package
- **Proto:** The protocol interface of message pack/unpack 
- **Codec:** Serialization interface for `Body`
- **XferPipe:** A series of pipelines to handle message data before transfer
- **XferFilter:** A interface to handle message data before transfer


## Message

The contents of every one message:

```go
// in .../teleport/socket package
type (
    // Message a socket message interface.
    Message interface {
        // Header is an operation interface of required message fields.
        // NOTE: Must be supported by Proto interface.
        Header

        // Body is an operation interface of optional message fields.
        // SUGGEST: For features complete, the protocol interface should support it.
        Body

        // XferPipe transfer filter pipe, handlers from outer-most to inner-most.
        // SUGGEST: The length can not be bigger than 255!
        XferPipe() *xfer.XferPipe

        // Size returns the size of message.
        // SUGGEST: For better statistics, Proto interfaces should support it.
        Size() uint32

        // SetSize sets the size of message.
        // If the size is too big, returns error.
        // SUGGEST: For better statistics, Proto interfaces should support it.
        SetSize(size uint32) error

        // Reset resets itself.
        Reset(settings ...MessageSetting)

        // Context returns the message handling context.
        Context() context.Context
        
        // String returns printing message information.
        String() string
    }

    // Header is an operation interface of required message fields.
    // NOTE: Must be supported by Proto interface.
    Header interface {
        // Seq returns the message sequence.
        Seq() int32
        // SetSeq sets the message sequence.
        SetSeq(int32)
        // Mtype returns the message type, such as CALL, REPLY, PUSH.
        Mtype() byte
        // Mtype sets the message type, such as CALL, REPLY, PUSH.
        SetMtype(byte)
        // ServiceMethod returns the serviec method.
        // SUGGEST: max len ≤ 255!
        ServiceMethod() string
        // SetServiceMethod sets the serviec method.
        // SUGGEST: max len ≤ 255!
        SetServiceMethod(string)
        // Meta returns the metadata.
        // SUGGEST: urlencoded string max len ≤ 65535!
        Meta() *utils.Args
    }

    // Body is an operation interface of optional message fields.
    // SUGGEST: For features complete, the protocol interface should support it.
    Body interface {
        // BodyCodec returns the body codec type id.
        BodyCodec() byte
        // SetBodyCodec sets the body codec type id.
        SetBodyCodec(bodyCodec byte)
        // Body returns the body object.
        Body() interface{}
        // SetBody sets the body object.
        SetBody(body interface{})
        // SetNewBody resets the function of geting body.
        //  NOTE: NewBodyFunc is only for reading form connection;
        SetNewBody(NewBodyFunc)
        // MarshalBody returns the encoding of body.
        // NOTE: when the body is a stream of bytes, no marshalling is done.
        MarshalBody() ([]byte, error)
        // UnmarshalBody unmarshals the encoded data to the body.
        // NOTE:
        //  seq, mtype, uri must be setted already;
        //  if body=nil, try to use newBodyFunc to create a new one;
        //  when the body is a stream of bytes, no unmarshalling is done.
        UnmarshalBody(bodyBytes []byte) error
    }

    // NewBodyFunc creates a new body by header,
    // and only for reading form connection.
    NewBodyFunc func(Header) interface{}
)

// in .../teleport/xfer package
type (
    // XferPipe transfer filter pipe, handlers from outer-most to inner-most.
    // NOTE: the length can not be bigger than 255!
    XferPipe struct {
        filters []XferFilter
    }
    // XferFilter handles byte stream of message when transfer.
    XferFilter interface {
        ID() byte
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
    // NOTE: Implementation specifications for Message interface should be complied with.
    Proto interface {
        // Version returns the protocol's id and name.
        Version() (byte, string)
        // Pack writes the Message into the connection.
        // NOTE: Make sure to write only once or there will be package contamination!
        Pack(Message) error
        // Unpack reads bytes from the connection to the Message.
        // NOTE: Concurrent unsafe!
        Unpack(Message) error
    }
    // IOWithReadBuffer implements buffered I/O with buffered reader.
    IOWithReadBuffer interface {
        io.ReadWriter
    }
    // ProtoFunc function used to create a custom Proto interface.
    ProtoFunc func(IOWithReadBuffer) Proto
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
{1 byte protocol version} # 6
{1 byte transfer pipe length}
{transfer pipe IDs}
# The following is handled data by transfer pipe
{1 bytes sequence length}
{sequence (HEX 36 string of int32)}
{1 byte message type} # e.g. CALL:1; REPLY:2; PUSH:3
{1 bytes service method length}
{service method}
{2 bytes status length}
{status(urlencoded)}
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
