package kcp

import (
	"crypto/tls"
	"net"

	kcp "github.com/xtaci/kcp-go/v5"
)

type (
	// UDPSession defines a KCP session implemented by UDP
	UDPSession = kcp.UDPSession
)

// DialAddrContext establishes a new KCP connection to a server.
// It uses a new UDP connection and closes this connection when the KCP session is closed.
// The hostname for SNI is taken from the given address.
func DialAddrContext(network string, laddr *net.UDPAddr, raddr string, tlsConf *tls.Config, dataShards, parityShards int) (net.Conn, error) {
	host, port, err := net.SplitHostPort(raddr)
	if err != nil {
		return nil, err
	}
	if host == "" {
		raddr = "127.0.0.1:" + port
	}
	addr, err := net.ResolveUDPAddr(network, raddr)
	if err != nil {
		return nil, err
	}
	udpConn, err := net.ListenUDP(network, laddr)
	if err != nil {
		return nil, err
	}
	conn, err := kcp.NewConn2(addr, nil, dataShards, parityShards, udpConn)
	if err != nil {
		return nil, err
	}
	if tlsConf != nil {
		return tls.Client(conn, tlsConf), nil
	}
	return conn, nil
}

// Listener defines a server which will be waiting to accept incoming connections
type Listener struct {
	*kcp.Listener
	tlsConf *tls.Config
	conn    net.PacketConn
}

var _ net.Listener = (*Listener)(nil)

// Accept implements the Accept method in the Listener interface; it waits for the next call and returns a generic Conn.
func (l *Listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	if l.tlsConf == nil {
		return conn, nil
	}
	return tls.Server(conn, l.tlsConf), nil
}

// PacketConn returns the net.PacketConn.
func (l *Listener) PacketConn() net.PacketConn {
	return l.conn
}

// ListenAddr announces on the local network address addr.
func ListenAddr(network, addr string, tlsConf *tls.Config, dataShards, parityShards int) (*Listener, error) {
	udpAddr, err := net.ResolveUDPAddr(network, addr)
	if err != nil {
		return nil, err
	}
	return ListenUDPAddr(network, udpAddr, tlsConf, dataShards, parityShards)
}

// ListenUDPAddr announces on the local network address udpAddr.
func ListenUDPAddr(network string, udpAddr *net.UDPAddr, tlsConf *tls.Config, dataShards, parityShards int) (*Listener, error) {
	var conn net.PacketConn
	conn, err := net.ListenUDP(network, udpAddr)
	if err != nil {
		return nil, err
	}
	return Listen(conn, tlsConf, dataShards, parityShards)
}

// Listen listens for KCP connections on a given net.PacketConn.
func Listen(conn net.PacketConn, tlsConf *tls.Config, dataShards, parityShards int) (*Listener, error) {
	lis, err := kcp.ServeConn(nil, dataShards, parityShards, conn)
	if err != nil {
		return nil, err
	}
	return &Listener{Listener: lis, tlsConf: tlsConf, conn: conn}, nil
}
