package quic

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	quic "github.com/quic-go/quic-go"
)

// DialAddrContext establishes a new QUIC connection to a server.
// It uses a new UDP connection and closes this connection when the QUIC session is closed.
// The hostname for SNI is taken from the given address.
func DialAddrContext(ctx context.Context, network string, laddr *net.UDPAddr, raddr string, tlsConf *tls.Config, config *quic.Config) (net.Conn, error) {
	host, port, err := net.SplitHostPort(raddr)
	if err != nil {
		return nil, err
	}
	if host == "" {
		raddr = "127.0.0.1:" + port
	}
	udpAddr, err := net.ResolveUDPAddr(network, raddr)
	if err != nil {
		return nil, err
	}
	udpConn, err := net.ListenUDP(network, laddr)
	if err != nil {
		return nil, err
	}
	sess, err := quic.Dial(ctx, udpConn, udpAddr, tlsConf, config)
	if err != nil {
		return nil, err
	}
	stream, err := sess.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return &Conn{
		sess:   sess,
		stream: stream,
	}, nil
}

// A Listener is a generic network listener for stream-oriented protocols.
//
// Multiple goroutines may invoke methods on a Listener simultaneously.
type Listener struct {
	lis  quic.Listener
	conn net.PacketConn
}

var _ net.Listener = (*Listener)(nil)

// ListenAddr announces on the local network address addr.
// The tls.Config must not be nil and must contain a certificate configuration.
// The quic.Config may be nil, in that case the default values will be used.
func ListenAddr(network, addr string, tlsConf *tls.Config, config *quic.Config) (*Listener, error) {
	udpAddr, err := net.ResolveUDPAddr(network, addr)
	if err != nil {
		return nil, err
	}
	return ListenUDPAddr(network, udpAddr, tlsConf, config)
}

// ListenUDPAddr announces on the local network address udpAddr.
// The tls.Config must not be nil and must contain a certificate configuration.
// The quic.Config may be nil, in that case the default values will be used.
func ListenUDPAddr(network string, udpAddr *net.UDPAddr, tlsConf *tls.Config, config *quic.Config) (*Listener, error) {
	conn, err := net.ListenUDP(network, udpAddr)
	if err != nil {
		return nil, err
	}
	return Listen(conn, tlsConf, config)
}

// Listen listens for QUIC connections on a given net.PacketConn.
// A single PacketConn only be used for a single call to Listen.
// The PacketConn can be used for simultaneous calls to Dial.
// QUIC connection IDs are used for demultiplexing the different connections.
// The tls.Config must not be nil and must contain a certificate configuration.
// The quic.Config may be nil, in that case the default values will be used.
func Listen(conn net.PacketConn, tlsConf *tls.Config, config *quic.Config) (*Listener, error) {
	if config == nil {
		config = &quic.Config{KeepAlivePeriod: time.Second * 15}
	}
	lis, err := quic.Listen(conn, tlsConf, config)
	if err != nil {
		return nil, err
	}
	return &Listener{
		lis:  *lis,
		conn: conn,
	}, nil
}

// PacketConn returns the net.PacketConn.
func (l *Listener) PacketConn() net.PacketConn {
	return l.conn
}

// Accept waits for and returns the next connection to the listener.
func (l *Listener) Accept() (net.Conn, error) {
	sess, err := l.lis.Accept(context.TODO())
	if err != nil {
		return nil, err
	}
	stream, err := sess.AcceptStream(context.TODO())
	if err != nil {
		return nil, err
	}
	return &Conn{
		sess:   sess,
		stream: stream,
	}, nil
}

// Close closes the listener PacketConn.
func (l *Listener) Close() error {
	return l.lis.Close()
}

// // Destroy destroys the listener.
// // Any blocked Accept operations will be unblocked and return errors.
// func (l *Listener) Destroy() error {
// 	return l.lis.Close()
// }

// Addr returns the listener's network address.
func (l *Listener) Addr() net.Addr {
	return l.lis.Addr()
}

// Conn is a QUIC network connection.
//
// Multiple goroutines may invoke methods on a Conn simultaneously.
type Conn struct {
	sess   quic.Connection
	stream quic.Stream
}

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c *Conn) Read(b []byte) (n int, err error) {
	return c.stream.Read(b)
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (c *Conn) Write(b []byte) (n int, err error) {
	return c.stream.Write(b)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *Conn) Close() error {
	err := c.stream.Close()
	if err != nil {
		c.sess.CloseWithError(1, err.Error())
		return err
	}
	return c.sess.CloseWithError(0, "")
}

// LocalAddr returns the local network address.
func (c *Conn) LocalAddr() net.Addr {
	return c.sess.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *Conn) RemoteAddr() net.Addr {
	return c.sess.RemoteAddr()
}

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
func (c *Conn) SetDeadline(t time.Time) error {
	return c.stream.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.stream.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.stream.SetWriteDeadline(t)
}
