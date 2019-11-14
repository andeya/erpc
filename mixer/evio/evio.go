package evio

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/mixer/evio/evio"
	"github.com/henrylee2cn/erpc/v6/utils"
)

// NewClient creates a evio client, equivalent to erpc.NewPeer.
func NewClient(cfg erpc.PeerConfig, globalLeftPlugin ...erpc.Plugin) erpc.Peer {
	return erpc.NewPeer(cfg, globalLeftPlugin...)
}

// Server a evio server
type Server struct {
	erpc.Peer
	cfg             erpc.PeerConfig
	events          evio.Events
	addr            string
	readBufferSize  int
	writeBufferSize int
}

// NewServer creates a evio server.
func NewServer(loops int, cfg erpc.PeerConfig, globalLeftPlugin ...erpc.Plugin) *Server {
	// globalLeftPlugin = append(globalLeftPlugin, new(wakeWritePlugin))
	p := erpc.NewPeer(cfg, globalLeftPlugin...)
	srv := &Server{
		Peer: p,
		cfg:  cfg,
		addr: fmt.Sprintf("%s://%s?reuseport=true", cfg.Network, cfg.ListenAddr()),
	}

	srv.events.NumLoops = loops
	// const delayDuration = time.Millisecond * 50
	// srv.events.Tick = func() (delay time.Duration, action evio.Action) {
	// 	return delayDuration, action
	// }

	srv.events.Serving = func(s evio.Server) (action evio.Action) {
		erpc.Printf("listen and serve (%s)", srv.addr)
		return
	}

	srv.events.Opened = func(c evio.Conn) (out []byte, opts evio.Options, action evio.Action) {
		stat := srv.serveConn(c)
		if !stat.OK() {
			erpc.Debugf("serve connection fail: %s", stat.String())
			action = evio.Close
		}
		opts.ReuseInputBuffer = true
		// opts.TCPKeepAlive = time.Minute * 60
		return
	}

	srv.events.Closed = func(c evio.Conn, err error) (action evio.Action) {
		if err != nil {
			erpc.Debugf("closed: %s: %s, error: %s", c.LocalAddr().String(), c.RemoteAddr().String(), err.Error())
		}
		con := c.Context().(*conn)
		con.sess.Close()
		close(con.out)
		return
	}

	srv.events.Data = func(c evio.Conn, in []byte) (out []byte, action evio.Action) {
		// defer func() {
		// 	if p := recover(); p != nil {
		// 		erpc.Errorf("[evio] Events.Data: %v", p)
		// 	}
		// }()
		con := c.Context().(*conn)
		if in != nil {
			buf := utils.AcquireByteBuffer()
			buf.Write(in)
			con.in <- buf
		}
		select {
		case out = <-con.out:
		default:
		}
		for len(con.out) > 0 {
			out = append(out, <-con.out...)
		}
		select {
		case <-con.closeSignal:
			action = evio.Close
			close(con.in)
		default:
		}
		return
	}
	return srv
}

// ListenAndServe turns on the listening service.
func (srv *Server) ListenAndServe(protoFunc ...erpc.ProtoFunc) error {
	switch srv.cfg.Network {
	default:
		return errors.New("Unsupport evio network, refer to the following: tcp, tcp4, tcp6, unix")
	case "tcp", "tcp4", "tcp6", "unix":
	}
	var isDefault bool
	srv.readBufferSize, isDefault = erpc.SocketReadBuffer()
	if isDefault {
		srv.readBufferSize = 4096
	}
	srv.writeBufferSize, isDefault = erpc.SocketWriteBuffer()
	if isDefault {
		srv.writeBufferSize = 4096
	}
	return evio.Serve(srv.events, srv.addr)
}

func (srv *Server) serveConn(evioConn evio.Conn) (stat *erpc.Status) {
	c := &conn{
		conn:        evioConn,
		events:      srv.events,
		closeSignal: make(chan struct{}),
		inBuf:       bytes.NewBuffer(make([]byte, 0, srv.readBufferSize)),
		in:          make(chan *utils.ByteBuffer, srv.readBufferSize/128),
		out:         make(chan []byte, srv.writeBufferSize/128),
	}
	if srv.TLSConfig() != nil {
		c.sess, stat = srv.Peer.ServeConn(tls.Server(c, srv.TLSConfig()))
	} else {
		c.sess, stat = srv.Peer.ServeConn(c)
	}
	// c.sess.Swap().Store(wakeWriteKey, c)
	evioConn.SetContext(c)
	return stat
}

// conn is a evio(evio) network connection.
//
// Multiple goroutines may invoke methods on a Conn simultaneously.
type conn struct {
	conn        evio.Conn
	events      evio.Events
	sess        erpc.Session
	inBuf       *bytes.Buffer
	in          chan *utils.ByteBuffer
	inLock      sync.Mutex
	out         chan []byte
	closeSignal chan struct{}
}

var _ net.Conn = new(conn)

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c *conn) Read(b []byte) (n int, err error) {
	n, err = c.inBuf.Read(b)
	if err == nil {
		return n, nil
	}
	buf, ok := <-c.in
	if !ok {
		return n, io.EOF
	}
	n2 := copy(b[n:], buf.B)
	c.inBuf.Write(buf.B[n2:])
	utils.ReleaseByteBuffer(buf)
	n += n2
	return n, nil
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (c *conn) Write(b []byte) (n int, err error) {
	cp := make([]byte, len(b))
	n = copy(cp, b)
	c.out <- cp
	c.conn.Wake()
	return n, nil
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *conn) Close() error {
	defer func() { recover() }()
	close(c.closeSignal)
	c.conn.Wake()
	return nil
}

// LocalAddr returns the local network address.
func (c *conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
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
func (c *conn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *conn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (c *conn) SetWriteDeadline(t time.Time) error {
	return nil
}
