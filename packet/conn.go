package packet

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

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

		// Write writes header and body to the connection.
		// Write can be made to time out and return an Error with Timeout() == true
		// after a fixed time limit; see SetDeadline and SetWriteDeadline.
		// Note: must be safe for concurrent use by multiple goroutines.
		Write(header *Header, body interface{}) (int, error)

		// ReadHeader reads header from the connection.
		// ReadHeader can be made to time out and return an Error with Timeout() == true
		// after a fixed time limit; see SetDeadline and SetReadDeadline.
		// Note: must use only one goroutine call.
		ReadHeader() (*Header, int, error)

		// ReadBody reads body from the connection.
		// ReadBody can be made to time out and return an Error with Timeout() == true
		// after a fixed time limit; see SetDeadline and SetReadDeadline.
		// Note: must use only one goroutine call, and it must be called after calling the ReadHeader().
		ReadBody(body interface{}) (int, error)

		// Close closes the connection.
		// Any blocked Read or Write operations will be unblocked and return errors.
		Close() error
	}
)

type conn struct {
	net.Conn
	bufWriter   *bufio.Writer
	bufReader   *bufio.Reader
	limitReader *utils.LimitedReader

	headerEncoder codec.Encoder
	headerDecoder codec.Decoder
	readedHeader  Header

	cacheWriter *bytes.Buffer
	encodeMap   map[string]codec.Encoder
	decodeMap   map[string]codec.Decoder

	gzipWriterMap map[int]*gzip.Writer
	gzipReader    *gzip.Reader

	writeMutex sync.Mutex // exclusive writer lock
}

// WrapConn wrap a net.Conn as a Conn
func WrapConn(c net.Conn) Conn {
	bufWriter := bufio.NewWriter(c)
	bufReader := bufio.NewReader(c)
	limitReader := utils.LimitReader(bufReader, 0)
	encodeMap := make(map[string]codec.Encoder)
	decodeMap := make(map[string]codec.Decoder)
	cacheWriter := bytes.NewBuffer(nil)
	return &conn{
		Conn:          c,
		bufWriter:     bufWriter,
		bufReader:     bufReader,
		limitReader:   limitReader,
		headerEncoder: codec.NewJsonEncoder(cacheWriter),
		headerDecoder: codec.NewJsonDecoder(limitReader),
		cacheWriter:   cacheWriter,
		encodeMap:     encodeMap,
		decodeMap:     decodeMap,
		gzipWriterMap: make(map[int]*gzip.Writer),
		gzipReader:    new(gzip.Reader),
	}
}

// Write writes header and body to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
// Note: must be safe for concurrent use by multiple goroutines.
func (c *conn) Write(header *Header, body interface{}) (int, error) {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()

	var n int
	err := binary.Write(c.bufWriter, binary.BigEndian, Magic)
	if err != nil {
		return n, err
	}
	n += len(Magic)

	// write header
	c.cacheWriter.Reset()
	n2, err := c.headerEncoder.Encode(header)
	if err != nil {
		return n, err
	}
	headerSize := uint32(n2)
	err = binary.Write(c.bufWriter, binary.BigEndian, headerSize)
	if err != nil {
		return n, err
	}
	n += 4

	n3, err := c.cacheWriter.WriteTo(c.bufWriter)
	n += int(n3)
	if err != nil {
		return n, err
	}

	// write body
	err = c.encodeBody(header, body)
	if err != nil {
		return n, err
	}

	bodySize := uint32(c.cacheWriter.Len())
	err = binary.Write(c.bufWriter, binary.BigEndian, bodySize)
	if err != nil {
		return n, err
	}
	n += 4

	n4, err := c.cacheWriter.WriteTo(c.bufWriter)
	n += int(n4)
	if err != nil {
		return n, err
	}

	err = c.bufWriter.Flush()
	return n, err
}

// ReadHeader reads header from the connection.
// ReadHeader can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
// Note: must use only one goroutine call.
func (c *conn) ReadHeader() (*Header, int, error) {
	var magic [len(Magic)]byte
	var n int
	err := binary.Read(c.bufReader, binary.BigEndian, &magic)
	if err != nil {
		return nil, 0, err
	}
	n += len(magic)
	if magic != Magic {
		return nil, n, fmt.Errorf("bad magic:%v", magic)
	}

	var headerSize uint32
	err = binary.Read(c.bufReader, binary.BigEndian, &headerSize)
	if err != nil {
		return nil, n, err
	}
	n += 4

	header := new(Header)
	c.limitReader.ResetLimit(int64(headerSize))
	n1, err := c.headerDecoder.Decode(header)
	n += n1
	if err != nil {
		return nil, n, err
	}
	c.readedHeader = *header
	return header, n, err
}

// DefaultCodec default codec name.
const DefaultCodec = "json"

// ReadBody reads body from the connection.
// ReadBody can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
// Note: must use only one goroutine call, and it must be called after calling the ReadHeader().
func (c *conn) ReadBody(body interface{}) (n int, err error) {
	var bodySize uint32
	err = binary.Read(c.bufReader, binary.BigEndian, &bodySize)
	if err != nil {
		return 0, err
	}
	n += 4

	c.limitReader.ResetLimit(int64(bodySize))

	name := c.readedHeader.Codec
	if len(name) == 0 {
		name = DefaultCodec
	}
	decoder, ok := c.decodeMap[name]
	if !ok {
		var decMaker codec.DecodeMaker
		decMaker, err = codec.GetDecodeMaker(name)
		if err != nil {
			return
		}
		if c.readedHeader.Gzip != gzip.NoCompression {
			err = c.gzipReader.Reset(c.limitReader)
			if err != nil {
				return
			}
			defer func() {
				err2 := c.gzipReader.Close()
				if err == nil {
					err = err2
				}
			}()
			decoder = decMaker(c.gzipReader)

		} else {
			decoder = decMaker(c.limitReader)
		}

		c.decodeMap[name] = decoder
	}

	n1, err := decoder.Decode(body)
	n += n1
	return
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *conn) Close() error {
	return c.Conn.Close()
}

func (c *conn) encodeBody(header *Header, body interface{}) (err error) {
	c.cacheWriter.Reset()

	name := header.Codec
	if len(name) == 0 {
		name = DefaultCodec
	}
	encoder, ok := c.encodeMap[name]
	if !ok {
		var encMaker codec.EncodeMaker
		encMaker, err = codec.GetEncodeMaker(name)
		if err != nil {
			return
		}
		level := int(header.Gzip)
		if level != gzip.NoCompression {
			gzipWriter, ok := c.gzipWriterMap[level]
			if !ok {
				gzipWriter, err = gzip.NewWriterLevel(c.cacheWriter, level)
				if err != nil {
					return
				}
				c.gzipWriterMap[level] = gzipWriter
			} else {
				gzipWriter.Reset(c.cacheWriter)
			}
			defer func() {
				err2 := gzipWriter.Close()
				if err == nil {
					err = err2
				}
			}()
			encoder = encMaker(gzipWriter)

		} else {
			encoder = encMaker(c.cacheWriter)
		}

		c.encodeMap[name] = encoder
	}

	_, err = encoder.Encode(body)
	return err
}
