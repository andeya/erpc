// Package socket provides a concise, powerful and high-performance TCP.
//
// Copyright 2017 HenryLee. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
package socket

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/errors"
)

type (
	// Socket is a generic stream-oriented network connection.
	// NOTE:
	//  Multiple goroutines may invoke methods on a Socket simultaneously.
	Socket interface {
		// ControlFD invokes f on the underlying connection's file
		// descriptor or handle.
		// The file descriptor fd is guaranteed to remain valid while
		// f executes but not after f returns.
		ControlFD(f func(fd uintptr)) error
		// LocalAddr returns the local network address.
		LocalAddr() net.Addr
		// RemoteAddr returns the remote network address.
		RemoteAddr() net.Addr
		// SetDeadline sets the read and write deadlines associated
		// with the connection. It is equivalent to calling both
		// SetReadDeadline and SetWriteDeadline.
		//
		// A deadline is an absolute time after which I/O operations
		// fail with a timeout (see type Error) instead of
		// blocking. The deadline applies to all future and pending
		// I/O, not just the immediately following call to Read or
		// Write. After a deadline has been exceeded, the connection
		// can be refreshed by setting a deadline in the future.
		//
		// An idle timeout can be implemented by repeatedly extending
		// the deadline after successful Read or Write calls.
		//
		// A zero value for t means I/O operations will not time out.
		SetDeadline(t time.Time) error
		// SetReadDeadline sets the deadline for future Read calls
		// and any currently-blocked Read call.
		// A zero value for t means Read will not time out.
		SetReadDeadline(t time.Time) error
		// SetWriteDeadline sets the deadline for future Write calls
		// and any currently-blocked Write call.
		// Even if write times out, it may return n > 0, indicating that
		// some of the data was successfully written.
		// A zero value for t means Write will not time out.
		SetWriteDeadline(t time.Time) error
		// WriteMessage writes header and body to the connection.
		// NOTE: must be safe for concurrent use by multiple goroutines.
		WriteMessage(message Message) error
		// ReadMessage reads header and body from the connection.
		// NOTE: must be safe for concurrent use by multiple goroutines.
		ReadMessage(message Message) error
		// Read reads data from the connection.
		// Read can be made to time out and return an Error with Timeout() == true
		// after a fixed time limit; see SetDeadline and SetReadDeadline.
		Read(b []byte) (n int, err error)
		// Write writes data to the connection.
		// Write can be made to time out and return an Error with Timeout() == true
		// after a fixed time limit; see SetDeadline and SetWriteDeadline.
		Write(b []byte) (n int, err error)
		// Close closes the connection socket.
		// Any blocked Read or Write operations will be unblocked and return errors.
		Close() error
		// Swap returns custom data swap of the socket.
		Swap() goutil.Map
		// SwapLen returns the amount of custom data of the socket.
		SwapLen() int
		// ID returns the socket id.
		ID() string
		// SetID sets the socket id.
		SetID(string)
		// Reset reset net.Conn and ProtoFunc.
		Reset(netConn net.Conn, protoFunc ...ProtoFunc)
		// Raw returns the raw net.Conn
		Raw() net.Conn
	}
	// UnsafeSocket has more unsafe methods than Socket interface.
	UnsafeSocket interface {
		Socket
		// RawLocked returns the raw net.Conn,
		// can be called in ProtoFunc.
		// NOTE:
		//  Make sure the external is locked before calling
		RawLocked() net.Conn
	}
	socket struct {
		net.Conn
		readerWithBuffer *bufio.Reader
		protocol         Proto
		id               string
		idMutex          sync.RWMutex
		swap             goutil.Map
		mu               sync.RWMutex
		curState         int32
		fromPool         bool
	}
)

const (
	normal      int32 = 0
	activeClose int32 = 1
)

var (
	_ net.Conn     = Socket(nil)
	_ UnsafeSocket = new(socket)
)

// ErrProactivelyCloseSocket proactively close the socket error.
var ErrProactivelyCloseSocket = errors.New("socket is closed proactively")

// GetSocket gets a Socket from pool, and reset it.
func GetSocket(c net.Conn, protoFunc ...ProtoFunc) Socket {
	s := socketPool.Get().(*socket)
	s.Reset(c, protoFunc...)
	return s
}

var socketPool = sync.Pool{
	New: func() interface{} {
		s := newSocket(nil, nil)
		s.fromPool = true
		return s
	},
}

var readerSize = 1024

// NewSocket wraps a net.Conn as a Socket.
func NewSocket(c net.Conn, protoFunc ...ProtoFunc) Socket {
	return newSocket(c, protoFunc)
}

func newSocket(c net.Conn, protoFuncs []ProtoFunc) *socket {
	var s = &socket{
		Conn:             c,
		readerWithBuffer: bufio.NewReaderSize(c, readerSize),
	}
	s.protocol = getProto(protoFuncs, s)
	s.initOptimize()
	return s
}

// Raw returns the raw net.Conn
func (s *socket) Raw() net.Conn {
	s.mu.RLock()
	conn := s.Conn
	s.mu.RUnlock()
	return conn
}

// RawLocked returns the raw net.Conn,
// can be called in ProtoFunc.
// NOTE:
//  Make sure the external is locked before calling
func (s *socket) RawLocked() net.Conn {
	return s.Conn
}

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (s *socket) Read(b []byte) (int, error) {
	return s.readerWithBuffer.Read(b)
}

// ControlFD invokes f on the underlying connection's file
// descriptor or handle.
// The file descriptor fd is guaranteed to remain valid while
// f executes but not after f returns.
func (s *socket) ControlFD(f func(fd uintptr)) error {
	syscallConn, ok := s.Raw().(syscall.Conn)
	if !ok {
		return syscall.EINVAL
	}
	ctrl, err := syscallConn.SyscallConn()
	if err != nil {
		return err
	}
	return ctrl.Control(f)
}

// WriteMessage writes header and body to the connection.
// WriteMessage can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
// NOTE:
//  For the byte stream type of body, write directly, do not do any processing;
//  Must be safe for concurrent use by multiple goroutines.
func (s *socket) WriteMessage(message Message) error {
	s.mu.RLock()
	protocol := s.protocol
	s.mu.RUnlock()
	err := protocol.Pack(message)
	if err != nil && s.isActiveClosed() {
		err = ErrProactivelyCloseSocket
	}
	return err
}

// ReadMessage reads header and body from the connection.
// NOTE:
//  For the byte stream type of body, read directly, do not do any processing;
//  Must be safe for concurrent use by multiple goroutines.
func (s *socket) ReadMessage(message Message) error {
	s.mu.RLock()
	protocol := s.protocol
	s.mu.RUnlock()
	return protocol.Unpack(message)
}

// Swap returns custom data swap of the socket.
func (s *socket) Swap() goutil.Map {
	if s.swap == nil {
		s.swap = goutil.RwMap()
	}
	return s.swap
}

// SwapLen returns the amount of custom data of the socket.
func (s *socket) SwapLen() int {
	if s.swap == nil {
		return 0
	}
	return s.swap.Len()
}

// ID returns the socket id.
func (s *socket) ID() string {
	s.idMutex.RLock()
	id := s.id
	if len(id) == 0 {
		id = s.RemoteAddr().String()
	}
	s.idMutex.RUnlock()
	return id
}

// SetID sets the socket id.
func (s *socket) SetID(id string) {
	s.idMutex.Lock()
	s.id = id
	s.idMutex.Unlock()
}

// Reset reset net.Conn and ProtoFunc.
func (s *socket) Reset(netConn net.Conn, protoFunc ...ProtoFunc) {
	atomic.StoreInt32(&s.curState, activeClose)
	s.mu.Lock()
	s.Conn = netConn
	s.readerWithBuffer.Discard(s.readerWithBuffer.Buffered())
	s.readerWithBuffer.Reset(netConn)
	s.protocol = getProto(protoFunc, s)
	s.SetID("")
	atomic.StoreInt32(&s.curState, normal)
	s.initOptimize()
	s.mu.Unlock()
}

// Close closes the connection socket.
// Any blocked Read or Write operations will be unblocked and return errors.
// If it is from 'GetSocket()' function(a pool), return itself to pool.
func (s *socket) Close() error {
	if s.isActiveClosed() {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isActiveClosed() {
		return nil
	}
	atomic.StoreInt32(&s.curState, activeClose)

	var err error
	if s.Conn != nil {
		err = s.Conn.Close()
	}
	if s.fromPool {
		s.Conn = nil
		s.swap = nil
		s.protocol = nil
		socketPool.Put(s)
	}
	return err
}

func (s *socket) isActiveClosed() bool {
	return atomic.LoadInt32(&s.curState) == activeClose
}

func (s *socket) initOptimize() {
	if c, ok := s.Conn.(ifaceSetKeepAlive); ok {
		if changeKeepAlive {
			c.SetKeepAlive(keepAlive)
		}
		if keepAlivePeriod >= 0 && keepAlive {
			c.SetKeepAlivePeriod(keepAlivePeriod)
		}
	}
	if c, ok := s.Conn.(ifaceSetBuffer); ok {
		if readBuffer >= 0 {
			c.SetReadBuffer(readBuffer)
		}
		if writeBuffer >= 0 {
			c.SetWriteBuffer(writeBuffer)
		}
	}
	if c, ok := s.Conn.(ifaceSetNoDelay); ok {
		if !noDelay {
			c.SetNoDelay(noDelay)
		}
	}
}

type (
	ifaceSetKeepAlive interface {
		// SetKeepAlive sets whether the operating system should send
		// keepalive messages on the connection.
		SetKeepAlive(keepalive bool) error
		// SetKeepAlivePeriod sets period between keep alives.
		SetKeepAlivePeriod(d time.Duration) error
	}
	ifaceSetBuffer interface {
		// SetReadBuffer sets the size of the operating system's
		// receive buffer associated with the connection.
		SetReadBuffer(bytes int) error
		// SetWriteBuffer sets the size of the operating system's
		// transmit buffer associated with the connection.
		SetWriteBuffer(bytes int) error
	}
	ifaceSetNoDelay interface {
		// SetNoDelay controls whether the operating system should delay
		// message transmission in hopes of sending fewer messages (Nagle's
		// algorithm).  The default is true (no delay), meaning that data is
		// sent as soon as possible after a Write.
		SetNoDelay(noDelay bool) error
	}
)

// Connection related system configuration
var (
	writeBuffer     int           = -1
	readBuffer      int           = -1
	changeKeepAlive bool          = false
	keepAlive       bool          = true
	keepAlivePeriod time.Duration = -1
	noDelay         bool          = true
)

// SetKeepAlive sets whether the operating system should send
// keepalive messages on the connection.
// NOTE: If have not called the function, the system defaults are used.
func SetKeepAlive(keepalive bool) {
	changeKeepAlive = true
	keepAlive = keepalive
}

// SetKeepAlivePeriod sets period between keep alives.
// NOTE: if d<0, don't change the value.
func SetKeepAlivePeriod(d time.Duration) {
	if d >= 0 {
		keepAlivePeriod = d
	} else {
		fmt.Println("socket: SetKeepAlivePeriod: invalid keepAlivePeriod:", d)
	}
}

// ReadBuffer returns the size of the operating system's
// receive buffer associated with the connection.
// NOTE: if using the system default value, bytes=-1 and isDefault=true.
func ReadBuffer() (bytes int, isDefault bool) {
	return readBuffer, readBuffer == -1
}

// SetReadBuffer sets the size of the operating system's
// receive buffer associated with the connection.
// NOTE: if bytes<0, don't change the value.
func SetReadBuffer(bytes int) {
	if bytes >= 0 {
		readBuffer = bytes
	} else {
		fmt.Println("socket: SetReadBuffer: invalid readBuffer size:", bytes)
	}
}

// WriteBuffer returns the size of the operating system's
// transmit buffer associated with the connection.
// NOTE: if using the system default value, bytes=-1 and isDefault=true.
func WriteBuffer() (bytes int, isDefault bool) {
	return writeBuffer, writeBuffer == -1
}

// SetWriteBuffer sets the size of the operating system's
// transmit buffer associated with the connection.
// NOTE: if bytes<0, don't change the value.
func SetWriteBuffer(bytes int) {
	if bytes >= 0 {
		writeBuffer = bytes
	} else {
		fmt.Println("socket: SetWriteBuffer: invalid writeBuffer size:", bytes)
	}
}

// SetNoDelay controls whether the operating system should delay
// message transmission in hopes of sending fewer messages (Nagle's
// algorithm).  The default is true (no delay), meaning that data is
// sent as soon as possible after a Write.
func SetNoDelay(_noDelay bool) {
	noDelay = _noDelay
}

func getProto(protoFuncs []ProtoFunc, rw IOWithReadBuffer) Proto {
	if len(protoFuncs) > 0 && protoFuncs[0] != nil {
		return protoFuncs[0](rw)
	}
	return defaultProtoFunc(rw)
}
