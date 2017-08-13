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
package socket

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/errors"

	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/utils"
)

// default codec.
var (
	DefaultCodecName = "json"
	DefaultCodec     codec.Codec
)

func init() {
	var err error
	DefaultCodec, err = codec.Get(DefaultCodecName)
	if err != nil {
		panic(err)
	}
}

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
	}
	socket struct {
		net.Conn
		id          string
		bufWriter   *utils.BufioWriter
		bufReader   *utils.BufioReader
		limitReader *utils.LimitedReader

		headerEncoder codec.Encoder
		headerDecoder codec.Decoder

		cacheWriter   *bytes.Buffer
		gzipWriterMap map[int]*gzip.Writer
		gzipReader    *gzip.Reader
		gzipEncodeMap map[string]*GzipEncoder // codecName:GzipEncoder
		gzipDecodeMap map[string]*GzipDecoder // codecName:GzipEncoder
		ctxPublic     goutil.Map
		writeMutex    sync.Mutex // exclusive writer lock
		readMutex     sync.Mutex // exclusive read lock
		fromPool      bool
	}
)

var _ net.Conn = Socket(nil)

// GetSocket gets a Socket from pool, and reset it.
func GetSocket(c net.Conn, id ...string) Socket {
	s := socketPool.Get().(*socket)
	s.Reset(c, id...)
	return s
}

var socketPool = sync.Pool{
	New: func() interface{} {
		s := NewSocket()
		s.fromPool = true
		return s
	},
}

// NewSocket wraps a net.Conn as a Socket.
func NewSocket() *socket {
	bufWriter := utils.NewBufioWriter(nil)
	bufReader := utils.NewBufioReader(nil)
	cacheWriter := bytes.NewBuffer(nil)
	limitReader := utils.LimitReader(bufReader, 0)
	headerEncoder, _ := codec.NewEncoder(DefaultCodecName, cacheWriter)
	headerDecoder, _ := codec.NewDecoder(DefaultCodecName, limitReader)
	return &socket{
		bufWriter:     bufWriter,
		bufReader:     bufReader,
		limitReader:   limitReader,
		headerEncoder: headerEncoder,
		headerDecoder: headerDecoder,
		cacheWriter:   cacheWriter,
		gzipWriterMap: make(map[int]*gzip.Writer),
		gzipReader:    new(gzip.Reader),
		gzipEncodeMap: make(map[string]*GzipEncoder),
		gzipDecodeMap: make(map[string]*GzipDecoder),
	}
}

// WritePacket writes header and body to the connection.
// WritePacket can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
// Note: must be safe for concurrent use by multiple goroutines.
func (s *socket) WritePacket(packet *Packet) error {
	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()
	s.bufWriter.ResetCount()

	// write header
	err := s.writeHeader(packet.Header)
	packet.HeaderLength = s.bufWriter.Count()
	packet.Length = packet.HeaderLength
	if err != nil {
		return err
	}
	defer func() {
		packet.BodyLength = s.bufWriter.Count() - packet.Length
		packet.Length = s.bufWriter.Count()
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
		err = s.writeCacheBody(packet.Header.Codec, int(packet.Header.Gzip), bo)
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
	}
	if err != nil {
		return err
	}
	return s.bufWriter.Flush()
}

func (s *socket) writeHeader(header *Header) error {
	s.cacheWriter.Reset()
	err := s.headerEncoder.Encode(header)
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

func (s *socket) writeCacheBody(codecName string, gzipLevel int, body interface{}) error {
	s.cacheWriter.Reset()
	if len(codecName) == 0 {
		codecName = DefaultCodecName
	}
	ge, err := s.getGzipEncoder(codecName)
	if err != nil {
		return err
	}
	return ge.Encode(gzipLevel, body)
}

// ReadPacket reads header and body from the connection.
// Note: must be safe for concurrent use by multiple goroutines.
func (s *socket) ReadPacket(packet *Packet) error {
	s.readMutex.Lock()
	defer s.readMutex.Unlock()
	var (
		hErr, bErr error
		b          interface{}
	)
	packet.HeaderLength, hErr = s.readHeader(packet.Header)
	if hErr == nil {
		b = packet.loadBody()
	}
	packet.BodyLength, bErr = s.readBody(packet.Header, b)
	packet.Length = packet.HeaderLength + packet.BodyLength
	return errors.Merge(hErr, bErr)
}

// readHeader reads header from the connection.
// readHeader can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
// Note: must use only one goroutine call.
func (s *socket) readHeader(header *Header) (int64, error) {
	s.bufReader.ResetCount()
	var headerSize uint32
	err := binary.Read(s.bufReader, binary.BigEndian, &headerSize)
	if err != nil {
		return s.bufReader.Count(), err
	}

	s.limitReader.ResetLimit(int64(headerSize))
	err = s.headerDecoder.Decode(header)
	if err != nil {
		return s.bufReader.Count(), err
	}
	return s.bufReader.Count(), err
}

// readBody reads body from the connection.
// readBody can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
// Note: must use only one goroutine call, and it must be called after calling the readHeader().
func (s *socket) readBody(header *Header, body interface{}) (int64, error) {
	s.bufReader.ResetCount()
	var bodySize uint32
	err := binary.Read(s.bufReader, binary.BigEndian, &bodySize)
	if err != nil {
		return s.bufReader.Count(), err
	}
	if bodySize == 0 {
		return s.bufReader.Count(), err
	}

	s.limitReader.ResetLimit(int64(bodySize))

	// read body
	switch bo := body.(type) {
	case nil:
		_, err = io.Copy(ioutil.Discard, s.limitReader)
		return s.bufReader.Count(), err

	case []byte:
		_, err = s.limitReader.Read(bo)
		if err != nil {
			return s.bufReader.Count(), err
		}
		_, err = io.Copy(ioutil.Discard, s.limitReader)
		return s.bufReader.Count(), err

	case *[]byte:
		*bo, err = ioutil.ReadAll(s.limitReader)
		return s.bufReader.Count(), err

	default:
		codecName := header.Codec
		if len(codecName) == 0 {
			codecName = DefaultCodecName
		}
		gd, err := s.getGzipDecoder(codecName)
		if err == nil {
			err = gd.Decode(int(header.Gzip), body)
		}
		return s.bufReader.Count(), err
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
	return s.id
}

// Reset reset net.Conn
func (s *socket) Reset(netConn net.Conn, id ...string) {
	if s.Conn != nil {
		s.Conn.Close()
	}
	var _id string
	if len(id) == 0 && netConn != nil {
		_id = netConn.RemoteAddr().String()
	} else {
		_id = id[0]
	}
	s.id = _id
	s.Conn = netConn
	s.bufReader.Reset(netConn)
	s.bufWriter.Reset(netConn)
}

// Close closes the connection socket.
// Any blocked Read or Write operations will be unblocked and return errors.
// If it is from 'GetSocket()' function(a pool), return itself to pool.
func (s *socket) Close() error {
	var errs []error
	errs = append(errs, s.Conn.Close())
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
	return errors.Merge(errs...)
}

func (s *socket) closeGzipReader() {
	defer func() {
		recover()
	}()
	s.gzipReader.Close()
}
