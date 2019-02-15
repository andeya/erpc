package quic

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/henrylee2cn/goutil/graceful"
	quic "github.com/lucas-clemente/quic-go"
)

// InheritedListen announces on the local network address laddr. The network net is "quic".
// It returns an inherited net.Listener for the matching network and address, or
// creates a new one using net.Listen.
func InheritedListen(laddr string, tlsConf *tls.Config, config *quic.Config) (net.Listener, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		return nil, err
	}
	return globalInheritQUIC.InheritedListen(udpAddr, tlsConf, config)
}

// SetInherited adds the files and envs to be inherited by the new process.
// Notes:
//  Only for reboot!
//  Windows system are not supported!
func SetInherited() error {
	return globalInheritQUIC.SetInherited()
}

const (
	// Used to indicate a graceful restart in the new process.
	envCountKey = "LISTEN_QUIC_FDS"
	// envCountKeyPrefix = envCountKey + "="
)

// In order to keep the working directory the same as when we started we record
// it at startup.
var originalWD, _ = os.Getwd()
var globalInheritQUIC = new(inheritQUIC)

// inheritQUIC provides the family of Listen functions and maintains the associated
// state. Typically you will have only once instance of inheritQUIC per application.
type inheritQUIC struct {
	inherited   []*net.UDPConn
	active      []*Listener
	mutex       sync.Mutex
	inheritOnce sync.Once

	// used in tests to override the default behavior of starting from fd 3.
	fdStart int
}

func (n *inheritQUIC) inherit() error {
	var retErr error
	n.inheritOnce.Do(func() {
		n.mutex.Lock()
		defer n.mutex.Unlock()
		countStr := os.Getenv(envCountKey)
		if countStr == "" {
			return
		}
		count, err := strconv.Atoi(countStr)
		if err != nil {
			retErr = fmt.Errorf("found invalid count value: %s=%s", envCountKey, countStr)
			return
		}

		// In tests this may be overridden.
		fdStart := n.fdStart
		if fdStart == 0 {
			// In normal operations if we are inheriting, the listeners will begin at
			// fd 3.
			fdStart = 3
		}

		for i := fdStart; i < fdStart+count; i++ {
			file := os.NewFile(uintptr(i), "listener")
			conn, err := net.FilePacketConn(file)
			if err != nil {
				file.Close()
				retErr = fmt.Errorf("error inheriting socket fd %d: %s", i, err)
				return
			}
			if err := file.Close(); err != nil {
				retErr = fmt.Errorf("error closing inherited socket fd %d: %s", i, err)
				return
			}
			if udpConn, ok := conn.(*net.UDPConn); ok {
				n.inherited = append(n.inherited, udpConn)
			}
		}
	})
	return retErr
}

// InheritedListen announces on the local network address laddr.
// It returns an inherited net.Listener for the matching address,
// or creates a new one.
func (n *inheritQUIC) InheritedListen(udpAddr *net.UDPAddr, tlsConf *tls.Config, config *quic.Config) (*Listener, error) {
	if err := n.inherit(); err != nil {
		return nil, err
	}

	n.mutex.Lock()
	defer n.mutex.Unlock()

	var udpConn *net.UDPConn

	// look for an inherited listener
	for i, conn := range n.inherited {
		if conn == nil { // we nil used inherited listeners
			continue
		}
		if isSameAddr(conn.LocalAddr(), udpAddr) {
			n.inherited[i] = nil
			udpConn = conn
		}
	}

	// make a fresh listener
	var l *Listener
	var err error
	if udpConn == nil {
		l, err = ListenUDPAddr(udpAddr, tlsConf, config)
	} else {
		l, err = Listen(udpConn, tlsConf, config)
	}
	if err != nil {
		return nil, err
	}
	n.active = append(n.active, l)
	return l, nil
}

// activeListeners returns a snapshot copy of the active listeners.
func (n *inheritQUIC) activeListeners() ([]*Listener, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	ls := make([]*Listener, len(n.active))
	copy(ls, n.active)
	return ls, nil
}

func isSameAddr(a1, a2 net.Addr) bool {
	if a1.Network() != a2.Network() {
		return false
	}
	a1s := a1.String()
	a2s := a2.String()
	if a1s == a2s {
		return true
	}

	// This allows for ipv6 vs ipv4 local addresses to compare as equal. This
	// scenario is common when listening on localhost.
	const ipv6prefix = "[::]"
	a1s = strings.TrimPrefix(a1s, ipv6prefix)
	a2s = strings.TrimPrefix(a2s, ipv6prefix)
	const ipv4prefix = "0.0.0.0"
	a1s = strings.TrimPrefix(a1s, ipv4prefix)
	a2s = strings.TrimPrefix(a2s, ipv4prefix)
	return a1s == a2s
}

// SetInherited adds the files and envs to be inherited by the new process.
// Notes:
//  Only for reboot!
//  Windows system are not supported!
func (n *inheritQUIC) SetInherited() error {
	listeners, err := n.activeListeners()
	if err != nil {
		return err
	}

	// Extract the fds from the listeners.
	var files = make([]*os.File, 0, len(listeners))
	for _, l := range listeners {
		f, err := l.PacketConn().(filer).File()
		if err != nil {
			return err
		}
		files = append(files, f)
	}

	graceful.AddInherited(files, []*graceful.Env{
		{K: envCountKey, V: strconv.Itoa(len(files))},
	})

	return nil
}

type filer interface {
	File() (*os.File, error)
}
