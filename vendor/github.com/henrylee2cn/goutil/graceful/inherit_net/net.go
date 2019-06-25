// Package inherit_net provides a family of Listen functions that either open a
// fresh connection or provide an inherited connection from when the process
// was started. The behave like their counterparts in the net package, but
// transparently provide support for graceful restarts without dropping
// connections. This is provided in a systemd socket activation compatible form
// to allow using socket activation.
//
// BUG: Doesn't handle closing of listeners.
package inherit_net

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/henrylee2cn/goutil/graceful"
)

// Listen announces on the local network address laddr. The network net must be
// a stream-oriented network: "tcp", "tcp4", "tcp6", "unix" or "unixpacket". It
// returns an inherited net.Listener for the matching network and address, or
// creates a new one using net.Listen.
func Listen(nett, laddr string) (net.Listener, error) {
	return globalInheritNet.Listen(nett, laddr)
}

// ListenTCP announces on the local network address laddr. The network net must
// be: "tcp", "tcp4" or "tcp6". It returns an inherited net.Listener for the
// matching network and address, or creates a new one using net.ListenTCP.
func ListenTCP(nett string, laddr *net.TCPAddr) (*net.TCPListener, error) {
	return globalInheritNet.ListenTCP(nett, laddr)
}

// ListenUnix announces on the local network address laddr. The network net
// must be a: "unix" or "unixpacket". It returns an inherited net.Listener for
// the matching network and address, or creates a new one using net.ListenUnix.
func ListenUnix(nett string, laddr *net.UnixAddr) (*net.UnixListener, error) {
	return globalInheritNet.ListenUnix(nett, laddr)
}

// Append append listener to inheritNet.active
func Append(ln net.Listener) error {
	return globalInheritNet.Append(ln)
}

// SetInherited adds the files and envs to be inherited by the new process.
// NOTE:
//  Only for reboot!
//  Windows system are not supported!
func SetInherited() error {
	return globalInheritNet.SetInherited()
}

const (
	// Used to indicate a graceful restart in the new process.
	envCountKey = "LISTEN_FDS"
	// envCountKeyPrefix = envCountKey + "="
)

// In order to keep the working directory the same as when we started we record
// it at startup.
var originalWD, _ = os.Getwd()
var globalInheritNet = new(inheritNet)

// inheritNet provides the family of Listen functions and maintains the associated
// state. Typically you will have only once instance of inheritNet per application.
type inheritNet struct {
	inherited   []net.Listener
	active      []net.Listener
	mutex       sync.Mutex
	inheritOnce sync.Once

	// used in tests to override the default behavior of starting from fd 3.
	fdStart int
}

func (n *inheritNet) inherit() error {
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
			l, err := net.FileListener(file)
			if err != nil {
				file.Close()
				retErr = fmt.Errorf("error inheriting socket fd %d: %s", i, err)
				return
			}
			if err := file.Close(); err != nil {
				retErr = fmt.Errorf("error closing inherited socket fd %d: %s", i, err)
				return
			}
			n.inherited = append(n.inherited, l)
		}
	})
	return retErr
}

// Listen announces on the local network address laddr. The network net must be
// a stream-oriented network: "tcp", "tcp4", "tcp6", "unix" or "unixpacket". It
// returns an inherited net.Listener for the matching network and address, or
// creates a new one using net.Listen.
func (n *inheritNet) Listen(nett, laddr string) (net.Listener, error) {
	switch nett {
	default:
		return nil, net.UnknownNetworkError(nett)
	case "tcp", "tcp4", "tcp6":
		addr, err := net.ResolveTCPAddr(nett, laddr)
		if err != nil {
			return nil, err
		}
		return n.ListenTCP(nett, addr)
	case "unix", "unixpacket", "invalid_unix_net_for_test":
		addr, err := net.ResolveUnixAddr(nett, laddr)
		if err != nil {
			return nil, err
		}
		return n.ListenUnix(nett, addr)
	}
}

// ListenTCP announces on the local network address laddr. The network net must
// be: "tcp", "tcp4" or "tcp6". It returns an inherited net.Listener for the
// matching network and address, or creates a new one using net.ListenTCP.
func (n *inheritNet) ListenTCP(nett string, laddr *net.TCPAddr) (*net.TCPListener, error) {
	if err := n.inherit(); err != nil {
		return nil, err
	}

	n.mutex.Lock()
	defer n.mutex.Unlock()

	// look for an inherited listener
	for i, l := range n.inherited {
		if l == nil { // we nil used inherited listeners
			continue
		}
		if isSameAddr(l.Addr(), laddr) {
			n.inherited[i] = nil
			n.active = append(n.active, l)
			return l.(*net.TCPListener), nil
		}
	}

	// make a fresh listener
	l, err := net.ListenTCP(nett, laddr)
	if err != nil {
		return nil, err
	}
	n.active = append(n.active, l)
	return l, nil
}

// ListenUnix announces on the local network address laddr. The network net
// must be a: "unix" or "unixpacket". It returns an inherited net.Listener for
// the matching network and address, or creates a new one using net.ListenUnix.
func (n *inheritNet) ListenUnix(nett string, laddr *net.UnixAddr) (*net.UnixListener, error) {
	if err := n.inherit(); err != nil {
		return nil, err
	}

	n.mutex.Lock()
	defer n.mutex.Unlock()

	// look for an inherited listener
	for i, l := range n.inherited {
		if l == nil { // we nil used inherited listeners
			continue
		}
		if isSameAddr(l.Addr(), laddr) {
			n.inherited[i] = nil
			n.active = append(n.active, l)
			return l.(*net.UnixListener), nil
		}
	}

	// make a fresh listener
	l, err := net.ListenUnix(nett, laddr)
	if err != nil {
		return nil, err
	}
	n.active = append(n.active, l)
	return l, nil
}

// Append append listener to inheritNet.active
func (n *inheritNet) Append(ln net.Listener) error {
	if err := n.inherit(); err != nil {
		return err
	}

	n.mutex.Lock()
	defer n.mutex.Unlock()

	// look for an inherited listener
	for i, l := range n.inherited {
		if l == nil { // we nil used inherited listeners
			continue
		}
		if isSameAddr(l.Addr(), ln.Addr()) {
			n.inherited[i] = nil
			n.active = append(n.active, l)
			return nil
		}
	}
	for _, l := range n.active {
		if l == nil { // we nil used inherited listeners
			continue
		}
		if isSameAddr(l.Addr(), ln.Addr()) {
			return fmt.Errorf("Re-register the listening port: network %s, address %s", ln.Addr().Network(), ln.Addr().String())
		}
	}
	n.active = append(n.active, ln)
	return nil
}

// activeListeners returns a snapshot copy of the active listeners.
func (n *inheritNet) activeListeners() ([]net.Listener, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	ls := make([]net.Listener, len(n.active))
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
// NOTE:
//  Only for reboot!
//  Windows system are not supported!
func (n *inheritNet) SetInherited() error {
	listeners, err := n.activeListeners()
	if err != nil {
		return err
	}

	// Extract the fds from the listeners.
	var files = make([]*os.File, 0, len(listeners))
	for _, l := range listeners {
		f, err := l.(filer).File()
		if err != nil {
			return err
		}
		files = append(files, f)
	}

	graceful.AddInherited(files, []*graceful.Env{
		{K: envCountKey, V: strconv.Itoa(len(listeners))},
	})

	return nil
}

type filer interface {
	File() (*os.File, error)
}
