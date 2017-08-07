// Copyright 2015-2017 HenryLee. All Rights Reserved.
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

package teleport

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	// "fmt"
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

type (
	// Conn is a generic stream-oriented network connection.
	//
	// Multiple goroutines may invoke methods on a Conn simultaneously.
	Conn interface {
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
		WritePacket(header *Header, body interface{}) (int64, error)

		// ReadHeader reads header from the connection.
		// ReadHeader can be made to time out and return an Error with Timeout() == true
		// after a fixed time limit; see SetDeadline and SetReadDeadline.
		// Note: must use only one goroutine call.
		ReadHeader() (*Header, int64, error)

		// ReadBody reads body from the connection.
		// ReadBody can be made to time out and return an Error with Timeout() == true
		// after a fixed time limit; see SetDeadline and SetReadDeadline.
		// Note: must use only one goroutine call, and it must be called after calling the ReadHeader().
		ReadBody(body interface{}) (int64, error)

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

		// Close closes the connection.
		// Any blocked Read or Write operations will be unblocked and return errors.
		Close() error

		// Public returns temporary public data of Conn.
		Public() goutil.Map
		// PublicLen returns the length of public data of Conn.
		PublicLen() int
	}
)

var _ net.Conn = Conn(nil)

type conn struct {
	net.Conn
	bufWriter   *utils.BufioWriter
	bufReader   *utils.BufioReader
	limitReader *utils.LimitedReader

	headerEncoder codec.Encoder
	headerDecoder codec.Decoder
	readedHeader  Header

	cacheWriter   *bytes.Buffer
	gzipWriterMap map[int]*gzip.Writer
	gzipReader    *gzip.Reader
	gzipEncodeMap map[string]*GzipEncoder // codecName:GzipEncoder
	gzipDecodeMap map[string]*GzipDecoder // codecName:GzipEncoder
	ctxPublic     goutil.Map
	writeMutex    sync.Mutex // exclusive writer lock
}

// WrapConn wrap a net.Conn as a Conn
func WrapConn(c net.Conn) Conn {
	obj := connPool.Get().(*conn)
	obj.Reset(c)
	return obj
}

var connPool = sync.Pool{
	New: func() interface{} {
		return newConn(nil)
	},
}

// newConn new a net.Conn as a Conn
func newConn(c net.Conn) *conn {
	bufWriter := utils.NewBufioWriter(c)
	bufReader := utils.NewBufioReader(c)
	cacheWriter := bytes.NewBuffer(nil)
	limitReader := utils.LimitReader(bufReader, 0)
	return &conn{
		Conn:          c,
		bufWriter:     bufWriter,
		bufReader:     bufReader,
		limitReader:   limitReader,
		headerEncoder: codec.NewJsonEncoder(cacheWriter),
		headerDecoder: codec.NewJsonDecoder(limitReader),
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
func (c *conn) WritePacket(header *Header, body interface{}) (int64, error) {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()
	c.bufWriter.ResetCount()

	// // write magic
	// err := binary.Write(c.bufWriter, binary.BigEndian, Magic)
	// if err != nil {
	// 	return 0, err
	// }
	// var n = len(Magic)

	// write header
	err := c.writeHeader(header)
	if err != nil {
		return c.bufWriter.Count(), err
	}

	// write body
	switch bo := body.(type) {
	case nil:
		err = binary.Write(c.bufWriter, binary.BigEndian, 0)
	case []byte:
		err = c.writeBytesBody(bo)
	case *[]byte:
		err = c.writeBytesBody(*bo)
	default:
		err = c.writeCacheBody(header.Codec, int(header.Gzip), body)
		if err != nil {
			return c.bufWriter.Count(), err
		}
		// write body to conn buffer
		bodySize := uint32(c.cacheWriter.Len())
		// println("bodySize========", bodySize)
		err = binary.Write(c.bufWriter, binary.BigEndian, bodySize)
		if err != nil {
			return c.bufWriter.Count(), err
		}
		_, err = c.cacheWriter.WriteTo(c.bufWriter)
	}
	if err != nil {
		return c.bufWriter.Count(), err
	}
	err = c.bufWriter.Flush()
	return c.bufWriter.Count(), err
}

func (c *conn) writeHeader(header *Header) error {
	c.cacheWriter.Reset()
	err := c.headerEncoder.Encode(header)
	if err != nil {
		return err
	}
	headerSize := uint32(c.cacheWriter.Len())
	err = binary.Write(c.bufWriter, binary.BigEndian, headerSize)
	if err != nil {
		return err
	}
	_, err = c.cacheWriter.WriteTo(c.bufWriter)
	return err
}

func (c *conn) writeBytesBody(body []byte) error {
	bodySize := uint32(len(body))
	err := binary.Write(c.bufWriter, binary.BigEndian, bodySize)
	if err != nil {
		return err
	}
	_, err = c.bufWriter.Write(body)
	return err
}

func (c *conn) writeCacheBody(codecName string, gzipLevel int, body interface{}) error {
	c.cacheWriter.Reset()
	if len(codecName) == 0 {
		codecName = DefaultCodec
	}
	ge, err := c.getGzipEncoder(codecName)
	if err != nil {
		return err
	}
	return ge.Encode(gzipLevel, body)
}

// ReadHeader reads header from the connection.
// ReadHeader can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
// Note: must use only one goroutine call.
func (c *conn) ReadHeader() (*Header, int64, error) {
	c.bufReader.ResetCount()
	// var magic [len(Magic)]byte
	// err := binary.Read(c.bufReader, binary.BigEndian, &magic)
	// if err != nil {
	// 	return nil, 0, err
	// }
	// var n = len(magic)
	// if magic != Magic {
	// 	return nil, n, fmt.Errorf("bad magic:%v", magic)
	// }

	var headerSize uint32
	err := binary.Read(c.bufReader, binary.BigEndian, &headerSize)
	if err != nil {
		return nil, c.bufReader.Count(), err
	}

	header := new(Header)
	c.limitReader.ResetLimit(int64(headerSize))
	err = c.headerDecoder.Decode(header)
	if err != nil {
		return nil, c.bufReader.Count(), err
	}
	c.readedHeader = *header
	return header, c.bufReader.Count(), err
}

// DefaultCodec default codec name.
const DefaultCodec = "json"

// ReadBody reads body from the connection.
// ReadBody can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
// Note: must use only one goroutine call, and it must be called after calling the ReadHeader().
func (c *conn) ReadBody(body interface{}) (int64, error) {
	c.bufReader.ResetCount()
	var bodySize uint32
	err := binary.Read(c.bufReader, binary.BigEndian, &bodySize)
	if err != nil {
		return c.bufReader.Count(), err
	}
	if bodySize == 0 {
		// println("bodySize == 0bodySize == 0bodySize == 0")
		return c.bufReader.Count(), err
	}

	c.limitReader.ResetLimit(int64(bodySize))

	// read body
	switch bo := body.(type) {
	case nil:
		_, err = io.Copy(ioutil.Discard, c.limitReader)
		return c.bufReader.Count(), err

	case []byte:
		_, err = c.limitReader.Read(bo)
		if err != nil {
			return c.bufReader.Count(), err
		}
		_, err = io.Copy(ioutil.Discard, c.limitReader)
		return c.bufReader.Count(), err

	case *[]byte:
		*bo, err = ioutil.ReadAll(c.limitReader)
		return c.bufReader.Count(), err

	default:
		codecName := c.readedHeader.Codec
		if len(codecName) == 0 {
			codecName = DefaultCodec
		}
		gd, err := c.getGzipDecoder(codecName)
		if err == nil {
			err = gd.Decode(int(c.readedHeader.Gzip), body)
		}
		return c.bufReader.Count(), err
	}
}

// Reset reset net.Conn
func (c *conn) Reset(netConn net.Conn) {
	if c.Conn != nil {
		c.Conn.Close()
	}
	c.Conn = netConn
	c.bufReader.Reset(netConn)
	c.bufWriter.Reset(netConn)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *conn) Close() error {
	var errs []error
	errs = append(errs, c.Conn.Close())
	c.Conn = nil
	c.bufReader.Reset(nil)
	c.bufWriter.Reset(nil)
	errs = append(errs, c.gzipReader.Close())
	for _, gz := range c.gzipWriterMap {
		errs = append(errs, gz.Close())
	}
	c.ctxPublic = nil
	connPool.Put(c)
	return errors.Merge(errs...)
}

// Public returns temporary public data of Conn.
func (c *conn) Public() goutil.Map {
	if c.ctxPublic == nil {
		c.ctxPublic = goutil.NormalMap()
	}
	return c.ctxPublic
}

// PublicLen returns the length of public data of Conn.
func (c *conn) PublicLen() int {
	if c.ctxPublic == nil {
		return 0
	}
	return c.ctxPublic.Len()
}

// var (
// 	Magic = [5]byte{'h', 'e', 'n', 'r', 'y'}
// )

// Header types
const (
	TypeRequest   int32 = 0
	TypeResponse  int32 = 1
	TypeAuth      int32 = 2
	TypeHeartbeat int32 = 3
)

// Header stauts codes
const (
	StatusSuccess int32 = 0
)
