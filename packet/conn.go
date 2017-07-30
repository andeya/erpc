package packet

import (
	"bufio"
	"bytes"
	"compress/gzip"
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
		// Close closes the connection.
		// Any blocked Read or Write operations will be unblocked and return errors.
		Close() error

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
		ReadHeader(header *Header) (int, error)

		// ReadBody reads body from the connection.
		// ReadBody can be made to time out and return an Error with Timeout() == true
		// after a fixed time limit; see SetDeadline and SetReadDeadline.
		ReadBody(body interface{}) (int, error)
	}
)

type conn struct {
	net.Conn
	bufWriter *bufio.Writer
	bufReader *bufio.Reader

	headerEncoder codec.Encoder
	headerDecoder codec.Decoder
	readedHeader  Header

	// bodyWriter io.Writer
	// bodyReader io.Reader

	bodyCacheWriter *bytes.Buffer
	encodeMap       map[string]codec.Encoder
	decodeMap       map[string]codec.Decoder
	encoderMutex    sync.Mutex
	decoderMutex    sync.Mutex

	gzipWriter *utils.GzipWriter
	gzipReader *gzip.Reader

	writeMutex sync.Mutex // exclusive writer lock
}

// WrapConn wrap a net.Conn as a Conn
func WrapConn(c net.Conn) Conn {
	bufWriter := bufio.NewWriter(c)
	bufReader := bufio.NewReader(c)
	encodeMap := make(map[string]codec.Encoder)
	decodeMap := make(map[string]codec.Decoder)
	bodyCacheWriter := bytes.NewBuffer(nil)
	return &conn{
		Conn:            c,
		bufWriter:       bufWriter,
		bufReader:       bufReader,
		headerEncoder:   codec.NewJsonEncoder(bufWriter),
		headerDecoder:   codec.NewJsonDecoder(bufReader),
		bodyCacheWriter: bodyCacheWriter,
		encodeMap:       encodeMap,
		decodeMap:       decodeMap,
		gzipWriter:      utils.NewGzipWriter(nil),
		gzipReader:      new(gzip.Reader),
	}
}

// Write writes header and body to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
// Note: must be safe for concurrent use by multiple goroutines.
func (c *conn) Write(header *Header, body interface{}) (int, error) {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()

	var err = c.encodeBody(header, body)
	if err != nil {
		return 0, err
	}

	header.Len = uint32(c.bodyCacheWriter.Len())
	n, err := c.headerEncoder.Encode(header)
	if err != nil {
		return n, err
	}

	n2, err := c.bodyCacheWriter.WriteTo(c.bufWriter)
	n += int(n2)
	if err != nil {
		return n, err
	}

	err = c.bufWriter.Flush()
	return n, err
}

// ReadHeader reads header from the connection.
// ReadHeader can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c *conn) ReadHeader(header *Header) (int, error) {
	n, err := c.headerDecoder.Decode(header)
	c.readedHeader = *header
	return n, err
}

// ReadBody reads body from the connection.
// ReadBody can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c *conn) ReadBody(body interface{}) (n int, err error) {
	bodyReader := utils.LimitReader(c.bufReader, int64(c.readedHeader.Len))
	name := c.readedHeader.Codec
	decoder, ok := c.decodeMap[name]
	if !ok {
		var decMaker codec.DecodeMaker
		decMaker, err = codec.GetDecodeMaker(name)
		if err != nil {
			return
		}
		if c.readedHeader.Gzip != gzip.NoCompression {
			err = c.gzipReader.Reset(bodyReader)
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
			decoder = decMaker(bodyReader)
		}

		c.decodeMap[name] = decoder
	}

	n, err = decoder.Decode(body)
	return
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *conn) Close() error {
	c.bufWriter.Flush()
	return c.Conn.Close()
}

func (c *conn) encodeBody(header *Header, body interface{}) (err error) {
	c.bodyCacheWriter.Reset()

	name := header.Codec
	encoder, ok := c.encodeMap[name]
	if !ok {
		var encMaker codec.EncodeMaker
		encMaker, err = codec.GetEncodeMaker(name)
		if err != nil {
			return
		}
		if header.Gzip != gzip.NoCompression {
			c.gzipWriter.ResetLevel(int(c.readedHeader.Gzip))
			c.gzipWriter.Reset(c.bodyCacheWriter)
			defer func() {
				err2 := c.gzipWriter.Close()
				if err == nil {
					err = err2
				}
			}()
			encoder = encMaker(c.gzipWriter)

		} else {
			encoder = encMaker(c.bodyCacheWriter)
		}

		c.encodeMap[name] = encoder
	}

	_, err = encoder.Encode(body)
	return err
}
