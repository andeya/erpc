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
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/errors"

	"github.com/henrylee2cn/teleport/codec"
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
		// WritePacket can be made to time out and return an Error with Timeout() == true
		// after a fixed time limit; see SetDeadline and SetWriteDeadline.
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
		// ChangeId changes the socket id.
		ChangeId(string)
	}
	socket struct {
		net.Conn
		id        string
		idMutex   sync.RWMutex
		bufWriter *utils.BufioWriter
		bufReader *utils.BufioReader

		cacheWriter *bytes.Buffer
		limitReader *utils.LimitedReader

		gzipWriterMap map[int]*gzip.Writer
		gzipReader    *gzip.Reader
		gzipEncodeMap map[string]*GzipEncoder // codecId:GzipEncoder
		gzipDecodeMap map[byte]*GzipDecoder   // codecId:GzipEncoder
		ctxPublic     goutil.Map
		writeMutex    sync.Mutex // exclusive writer lock
		readMutex     sync.Mutex // exclusive read lock
		curState      int32
		fromPool      bool
	}
)

const (
	idle        int32 = 0
	activeClose int32 = 1
)

var _ net.Conn = Socket(nil)

// ErrProactivelyCloseSocket proactively close the socket error.
var ErrProactivelyCloseSocket = errors.New("socket: Proactively close the socket")

// GetSocket gets a Socket from pool, and reset it.
func GetSocket(c net.Conn, id ...string) Socket {
	s := socketPool.Get().(*socket)
	s.Reset(c, id...)
	return s
}

var socketPool = sync.Pool{
	New: func() interface{} {
		s := NewSocket(nil)
		s.fromPool = true
		return s
	},
}

// NewSocket wraps a net.Conn as a Socket.
func NewSocket(c net.Conn, id ...string) *socket {
	bufWriter := utils.NewBufioWriter(c)
	bufReader := utils.NewBufioReader(c)
	cacheWriter := bytes.NewBuffer(nil)
	limitReader := utils.LimitReader(bufReader, 0)
	var s = &socket{
		id:            getId(c, id),
		Conn:          c,
		bufWriter:     bufWriter,
		bufReader:     bufReader,
		cacheWriter:   cacheWriter,
		limitReader:   limitReader,
		gzipWriterMap: make(map[int]*gzip.Writer),
		gzipReader:    new(gzip.Reader),
		gzipEncodeMap: make(map[string]*GzipEncoder),
		gzipDecodeMap: make(map[byte]*GzipDecoder),
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
	defer func() {
		if err != nil && atomic.LoadInt32(&s.curState) == activeClose {
			err = ErrProactivelyCloseSocket
		}
		s.writeMutex.Unlock()
	}()
	s.bufWriter.ResetCount()

	if len(packet.HeaderCodec) == 0 {
		packet.HeaderCodec = defaultHeaderCodec.Name()
	}

	// write header
	err = s.writeHeader(packet.HeaderCodec, packet.Header)
	packet.HeaderLength = s.bufWriter.Count()
	packet.Length = packet.HeaderLength
	packet.BodyLength = 0
	if err != nil {
		return err
	}
	defer func() {
		packet.Length = s.bufWriter.Count()
		packet.BodyLength = packet.Length - packet.HeaderLength
	}()
	// write body
	switch bo := packet.Body.(type) {
	case nil:
		err = binary.Write(s.bufWriter, binary.BigEndian, uint32(0))
	case []byte:
		err = s.writeBytesBody(bo)
	case *[]byte:
		err = s.writeBytesBody(*bo)
	default:
		if len(packet.BodyCodec) == 0 {
			packet.BodyCodec = defaultBodyCodec.Name()
		}
		err = s.writeBody(packet.BodyCodec, int(packet.Header.Gzip), bo)
	}
	if err != nil {
		return err
	}
	return s.bufWriter.Flush()
}

func (s *socket) writeHeader(codecName string, header *Header) error {
	s.cacheWriter.Reset()
	ge, err := s.getGzipEncoder(codecName)
	if err != nil {
		return err
	}
	err = binary.Write(s.cacheWriter, binary.BigEndian, ge.Id())
	if err != nil {
		return err
	}
	err = ge.Encode(gzip.NoCompression, header)
	if err != nil {
		return err
	}
	headerSize := uint32(s.cacheWriter.Len())
	err = binary.Write(s.bufWriter, binary.BigEndian, headerSize)
	if err != nil {
		return err
	}
	_, err = s.cacheWriter.WriteTo(s.bufWriter)
	return err
}

func (s *socket) writeBytesBody(body []byte) error {
	bodySize := uint32(len(body))
	err := binary.Write(s.bufWriter, binary.BigEndian, bodySize)
	if err != nil {
		return err
	}
	_, err = s.bufWriter.Write(body)
	return err
}

func (s *socket) writeBody(codecName string, gzipLevel int, body interface{}) error {
	s.cacheWriter.Reset()
	ge, err := s.getGzipEncoder(codecName)
	if err != nil {
		return err
	}
	err = binary.Write(s.cacheWriter, binary.BigEndian, ge.Id())
	if err != nil {
		return err
	}
	err = ge.Encode(gzipLevel, body)
	if err != nil {
		return err
	}
	// write body to socket buffer
	bodySize := uint32(s.cacheWriter.Len())
	err = binary.Write(s.bufWriter, binary.BigEndian, bodySize)
	if err != nil {
		return err
	}
	_, err = s.cacheWriter.WriteTo(s.bufWriter)
	return err
}

// ReadPacket reads header and body from the connection.
// Note:
//  For the byte stream type of body, read directly, do not do any processing;
//  Must be safe for concurrent use by multiple goroutines.
func (s *socket) ReadPacket(packet *Packet) error {
	s.readMutex.Lock()
	defer s.readMutex.Unlock()
	var (
		hErr, bErr error
		b          interface{}
	)

	packet.HeaderLength, packet.HeaderCodec, hErr = s.readHeader(packet.Header)
	if hErr == nil {
		b = packet.loadBody()
	} else {
		if hErr == io.EOF || hErr == io.ErrUnexpectedEOF {
			packet.Length = packet.HeaderLength
			packet.BodyLength = 0
			packet.BodyCodec = ""
			return hErr
		} else if atomic.LoadInt32(&s.curState) == activeClose {
			packet.Length = packet.HeaderLength
			packet.BodyLength = 0
			packet.BodyCodec = ""
			return ErrProactivelyCloseSocket
		}
	}

	packet.BodyLength, packet.BodyCodec, bErr = s.readBody(int(packet.Header.Gzip), b)
	packet.Length = packet.HeaderLength + packet.BodyLength
	if atomic.LoadInt32(&s.curState) == activeClose {
		return ErrProactivelyCloseSocket
	}
	return bErr
}

// readHeader reads header from the connection.
// readHeader can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
// Note: must use only one goroutine call.
func (s *socket) readHeader(header *Header) (int64, string, error) {
	s.bufReader.ResetCount()
	var (
		headerSize uint32
		codecId    = codec.NilCodecId
	)

	err := binary.Read(s.bufReader, binary.BigEndian, &headerSize)
	if err != nil {
		return s.bufReader.Count(), "", err
	}

	s.limitReader.ResetLimit(int64(headerSize))

	err = binary.Read(s.limitReader, binary.BigEndian, &codecId)

	if err != nil {
		return s.bufReader.Count(), getCodecName(codecId), err
	}

	gd, err := s.getGzipDecoder(codecId)
	if err != nil {
		return s.bufReader.Count(), getCodecName(codecId), err
	}
	err = gd.Decode(gzip.NoCompression, header)
	return s.bufReader.Count(), gd.Name(), err
}

// readBody reads body from the connection.
// readBody can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
// Note: must use only one goroutine call, and it must be called after calling the readHeader().
func (s *socket) readBody(gzipLevel int, body interface{}) (int64, string, error) {
	s.bufReader.ResetCount()
	var (
		bodySize uint32
		codecId  = codec.NilCodecId
	)

	err := binary.Read(s.bufReader, binary.BigEndian, &bodySize)
	if err != nil {
		return s.bufReader.Count(), "", err
	}
	if bodySize == 0 {
		return s.bufReader.Count(), "", err
	}

	s.limitReader.ResetLimit(int64(bodySize))

	// read body
	switch bo := body.(type) {
	case nil:
		_, err = io.Copy(ioutil.Discard, s.limitReader)
		return s.bufReader.Count(), "", err

	case []byte:
		_, err = s.limitReader.Read(bo)
		if err != nil {
			return s.bufReader.Count(), "", err
		}
		_, err = io.Copy(ioutil.Discard, s.limitReader)
		return s.bufReader.Count(), "", err

	case *[]byte:
		*bo, err = ioutil.ReadAll(s.limitReader)
		return s.bufReader.Count(), "", err

	default:
		err = binary.Read(s.limitReader, binary.BigEndian, &codecId)
		if err != nil {
			return s.bufReader.Count(), getCodecName(codecId), err
		}
		gd, err := s.getGzipDecoder(codecId)
		if err != nil {
			return s.bufReader.Count(), getCodecName(codecId), err
		}
		err = gd.Decode(gzipLevel, body)
		return s.bufReader.Count(), gd.Name(), err
	}
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
	s.idMutex.RUnlock()
	return id
}

// ChangeId changes the socket id.
func (s *socket) ChangeId(id string) {
	s.idMutex.Lock()
	s.id = id
	s.idMutex.Unlock()
}

// Reset reset net.Conn
func (s *socket) Reset(netConn net.Conn, id ...string) {
	atomic.StoreInt32(&s.curState, activeClose)
	if s.Conn != nil {
		s.Conn.Close()
	}
	s.readMutex.Lock()
	s.writeMutex.Lock()
	s.ChangeId(getId(netConn, id))
	s.Conn = netConn
	s.bufReader.Reset(netConn)
	s.bufWriter.Reset(netConn)
	atomic.StoreInt32(&s.curState, idle)
	s.readMutex.Unlock()
	s.writeMutex.Unlock()
}

// Close closes the connection socket.
// Any blocked Read or Write operations will be unblocked and return errors.
// If it is from 'GetSocket()' function(a pool), return itself to pool.
func (s *socket) Close() error {
	atomic.StoreInt32(&s.curState, activeClose)
	var errs []error
	if s.Conn != nil {
		errs = append(errs, s.Conn.Close())
	}
	s.readMutex.Lock()
	s.writeMutex.Lock()
	s.Conn = nil
	s.bufReader.Reset(nil)
	s.bufWriter.Reset(nil)
	s.closeGzipReader()
	for _, gz := range s.gzipWriterMap {
		errs = append(errs, gz.Close())
	}
	s.ctxPublic = nil
	if s.fromPool {
		socketPool.Put(s)
	}
	s.readMutex.Unlock()
	s.writeMutex.Unlock()
	return errors.Merge(errs...)
}

func (s *socket) closeGzipReader() {
	defer func() {
		recover()
	}()
	s.gzipReader.Close()
}

func getId(c net.Conn, ids []string) string {
	var id string
	if len(ids) > 0 && len(ids[0]) > 0 {
		id = ids[0]
	} else if c != nil {
		id = c.RemoteAddr().String()
	}
	return id
}
