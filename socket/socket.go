// Socket package provides a concise, powerful and high-performance TCP socket.
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
//
// Packet protocol:
//  HeaderLength | HeaderCodecId | Header | BodyLength | BodyCodecId | Body
//
// Notes:
//  HeaderLength: uint32, 4 bytes, big endian
//  BodyLength: uint32, 4 bytes, big endian
//  HeaderCodecId: uint8, 1 byte
//  BodyCodecId: uint8, 1 byte
//
package socket

import (
	"compress/gzip"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/errors"
	"github.com/henrylee2cn/teleport/utils"
)

type (
	// Socket is a generic stream-oriented network connection.
	//
	// Multiple goroutines may invoke methods on a Socket simultaneously.
	Socket interface {
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

		// WritePacket writes header and body to the connection.
		// Note: must be safe for concurrent use by multiple goroutines.
		WritePacket(packet *Packet) error

		// ReadPacket reads header and body from the connection.
		// Note: must be safe for concurrent use by multiple goroutines.
		ReadPacket(packet *Packet) error

		// Read reads data from the connection.
		// Read can be made to time out and return an Error with Timeout() == true
		// after a fixed time limit; see SetDeadline and SetReadDeadline.
		Read(b []byte) (n int, err error)

		// Write writes data to the connection.
		// Write can be made to time out and return an Error with Timeout() == true
		// after a fixed time limit; see SetDeadline and SetWriteDeadline.
		Write(b []byte) (n int, err error)

		// Reset reset net.Conn
		// Reset(net.Conn)

		// Close closes the connection socket.
		// Any blocked Read or Write operations will be unblocked and return errors.
		Close() error

		// Public returns temporary public data of Socket.
		Public() goutil.Map
		// PublicLen returns the length of public data of Socket.
		PublicLen() int
		// Id returns the socket id.
		Id() string
		// SetId sets the socket id.
		SetId(string)
	}
	socket struct {
		net.Conn
		id        string
		idMutex   sync.RWMutex
		ctxPublic goutil.Map
		protocol  Protocol
		curState  int32
		fromPool  bool

		// about write
		bufioWriter    *utils.BufioWriter
		codecWriterMap map[string]*CodecWriter // codecId:CodecWriter
		gzipWriterMap  map[int]*gzip.Writer
		writeMutex     sync.Mutex // exclusive writer lock

		// about read
		bufioReader    *utils.BufioReader
		codecReaderMap map[byte]*CodecReader // codecId:CodecReader
		gzipReader     *gzip.Reader
		readMutex      sync.Mutex // exclusive read lock
	}
)

const (
	normal      int32 = 0
	activeClose int32 = 1
)

var _ net.Conn = Socket(nil)

// ErrProactivelyCloseSocket proactively close the socket error.
var ErrProactivelyCloseSocket = errors.New("socket is closed proactively")

// GetSocket gets a Socket from pool, and reset it.
func GetSocket(c net.Conn, protocol ...Protocol) Socket {
	s := socketPool.Get().(*socket)
	s.Reset(c, protocol...)
	return s
}

var socketPool = sync.Pool{
	New: func() interface{} {
		s := newSocket(nil, nil)
		s.fromPool = true
		return s
	},
}

// NewSocket wraps a net.Conn as a Socket.
func NewSocket(c net.Conn, protocol ...Protocol) Socket {
	return newSocket(c, protocol)
}

func newSocket(c net.Conn, protocols []Protocol) *socket {
	bufioWriter := utils.NewBufioWriter(c)
	bufioReader := utils.NewBufioReader(c)
	var s = &socket{
		protocol:       getProtocol(protocols),
		Conn:           c,
		bufioWriter:    bufioWriter,
		bufioReader:    bufioReader,
		gzipWriterMap:  make(map[int]*gzip.Writer),
		gzipReader:     new(gzip.Reader),
		codecWriterMap: make(map[string]*CodecWriter),
		codecReaderMap: make(map[byte]*CodecReader),
	}
	return s
}

// WritePacket writes header and body to the connection.
// WritePacket can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
// Note:
//  For the byte stream type of body, write directly, do not do any processing;
//  Must be safe for concurrent use by multiple goroutines.
func (s *socket) WritePacket(packet *Packet) (err error) {
	s.writeMutex.Lock()
	if len(packet.HeaderCodec) == 0 {
		packet.HeaderCodec = defaultHeaderCodec.Name()
	}
	if len(packet.BodyCodec) == 0 {
		packet.BodyCodec = defaultBodyCodec.Name()
	}
	defer func() {
		if err != nil && s.isActiveClosed() {
			err = ErrProactivelyCloseSocket
		}
		s.writeMutex.Unlock()
	}()
	s.bufioWriter.ResetCount()
	return s.protocol.WritePacket(
		packet,
		s.bufioWriter,
		s.getCodecWriter,
		s.isActiveClosed,
	)
}

// ReadPacket reads header and body from the connection.
// Note:
//  For the byte stream type of body, read directly, do not do any processing;
//  Must be safe for concurrent use by multiple goroutines.
func (s *socket) ReadPacket(packet *Packet) error {
	s.readMutex.Lock()
	defer s.readMutex.Unlock()
	return s.protocol.ReadPacket(
		packet,
		packet.bodyAdapter,
		s.bufioReader,
		s.makeCodecReader,
		s.isActiveClosed,
		checkReadLimit,
	)
}

// Public returns temporary public data of Socket.
func (s *socket) Public() goutil.Map {
	if s.ctxPublic == nil {
		s.ctxPublic = goutil.RwMap()
	}
	return s.ctxPublic
}

// PublicLen returns the length of public data of Socket.
func (s *socket) PublicLen() int {
	if s.ctxPublic == nil {
		return 0
	}
	return s.ctxPublic.Len()
}

// Id returns the socket id.
func (s *socket) Id() string {
	s.idMutex.RLock()
	id := s.id
	if len(id) == 0 {
		id = s.RemoteAddr().String()
	}
	s.idMutex.RUnlock()
	return id
}

// SetId sets the socket id.
func (s *socket) SetId(id string) {
	s.idMutex.Lock()
	s.id = id
	s.idMutex.Unlock()
}

// Reset reset net.Conn
func (s *socket) Reset(netConn net.Conn, protocol ...Protocol) {
	atomic.StoreInt32(&s.curState, activeClose)
	if s.Conn != nil {
		s.Conn.Close()
	}
	s.readMutex.Lock()
	s.writeMutex.Lock()
	s.Conn = netConn
	s.bufioReader.Reset(netConn)
	s.bufioWriter.Reset(netConn)
	s.SetId("")
	s.protocol = getProtocol(protocol)
	atomic.StoreInt32(&s.curState, normal)
	s.readMutex.Unlock()
	s.writeMutex.Unlock()
}

// Close closes the connection socket.
// Any blocked Read or Write operations will be unblocked and return errors.
// If it is from 'GetSocket()' function(a pool), return itself to pool.
func (s *socket) Close() error {
	if s.isActiveClosed() {
		return nil
	}
	atomic.StoreInt32(&s.curState, activeClose)

	var errs []error
	if s.Conn != nil {
		errs = append(errs, s.Conn.Close())
	}

	s.readMutex.Lock()
	s.writeMutex.Lock()
	defer func() {
		s.readMutex.Unlock()
		s.writeMutex.Unlock()
	}()

	if s.isActiveClosed() {
		return nil
	}

	s.closeGzipReader()
	for _, gz := range s.gzipWriterMap {
		errs = append(errs, gz.Close())
	}
	if s.fromPool {
		s.Conn = nil
		s.ctxPublic = nil
		s.bufioReader.Reset(nil)
		s.bufioWriter.Reset(nil)
		socketPool.Put(s)
	}
	return errors.Merge(errs...)
}

func (s *socket) isActiveClosed() bool {
	return atomic.LoadInt32(&s.curState) == activeClose
}

func (s *socket) closeGzipReader() {
	defer func() {
		recover()
	}()
	s.gzipReader.Close()
}

func getProtocol(protocols []Protocol) Protocol {
	if len(protocols) > 0 {
		return protocols[0]
	} else {
		return defaultProtocol
	}
}

// func getId(c net.Conn, ids []string) string {
// 	var id string
// 	if len(ids) > 0 && len(ids[0]) > 0 {
// 		id = ids[0]
// 	} else if c != nil {
// 		id = c.RemoteAddr().String()
// 	}
// 	return id
// }
