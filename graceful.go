// Copyright 2015-2018 HenryLee. All Rights Reserved.
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

package tp

import (
	"crypto/tls"
	"encoding/json"
	"net"
	"os"
	"sync"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/errors"
	"github.com/henrylee2cn/goutil/graceful"
	"github.com/henrylee2cn/goutil/graceful/inherit_net"
	"github.com/henrylee2cn/teleport/quic"
)

var peers = struct {
	list map[*peer]struct{}
	rwmu sync.RWMutex
}{
	list: make(map[*peer]struct{}),
}

func addPeer(p *peer) {
	peers.rwmu.Lock()
	peers.list[p] = struct{}{}
	peers.rwmu.Unlock()
}

func deletePeer(p *peer) {
	peers.rwmu.Lock()
	delete(peers.list, p)
	peers.rwmu.Unlock()
}

func shutdown() error {
	peers.rwmu.RLock()
	var (
		list  []*peer
		count int
		errCh = make(chan error, len(list))
	)
	for p := range peers.list {
		list = append(list, p)
	}
	peers.rwmu.RUnlock()
	for _, p := range list {
		count++
		go func(peer *peer) {
			errCh <- peer.Close()
		}(p)
	}
	var err error
	for i := 0; i < count; i++ {
		err = errors.Merge(err, <-errCh)
	}
	close(errCh)
	return err
}

func init() {
	graceful.SetLog(logger)
	initParentLaddrList()
	SetShutdown(5*time.Second, nil, nil)
}

// GraceSignal open graceful shutdown or reboot signal.
func GraceSignal() {
	graceful.GraceSignal()
}

var (
	// FirstSweep is first executed.
	// Usage: share github.com/henrylee2cn/goutil/graceful with other project.
	FirstSweep func() error
	// BeforeExiting is executed before process exiting.
	// Usage: share github.com/henrylee2cn/goutil/graceful with other project.
	BeforeExiting func() error
)

// SetShutdown sets the function which is called after the process shutdown,
// and the time-out period for the process shutdown.
// If 0<=timeout<5s, automatically use 'MinShutdownTimeout'(5s).
// If timeout<0, indefinite period.
// 'firstSweep' is first executed.
// 'beforeExiting' is executed before process exiting.
func SetShutdown(timeout time.Duration, firstSweep, beforeExiting func() error) {
	if firstSweep == nil {
		firstSweep = func() error { return nil }
	}
	if beforeExiting == nil {
		beforeExiting = func() error { return nil }
	}
	FirstSweep = func() error {
		setParentLaddrList()
		return errors.Merge(firstSweep(), inherit_net.SetInherited(), quic.SetInherited())
	}
	BeforeExiting = func() error {
		return errors.Merge(shutdown(), beforeExiting())
	}
	graceful.SetShutdown(timeout, FirstSweep, BeforeExiting)
}

// Shutdown closes all the frame process gracefully.
// Parameter timeout is used to reset time-out period for the process shutdown.
func Shutdown(timeout ...time.Duration) {
	graceful.Shutdown(timeout...)
}

// Reboot all the frame process gracefully.
// NOTE: Windows system are not supported!
func Reboot(timeout ...time.Duration) {
	graceful.Reboot(timeout...)
}

var testTLSConfig = GenerateTLSConfigForServer()

// NewInheritedListener creates a new inherited listener.
func NewInheritedListener(network, laddr string, tlsConfig *tls.Config) (lis net.Listener, err error) {
	host, port, err := net.SplitHostPort(laddr)
	if err != nil {
		return nil, err
	}
	if port == "0" {
		laddr = popParentLaddr(network, host, laddr)
	}

	if network == "quic" {
		if tlsConfig == nil {
			tlsConfig = testTLSConfig
		}
		lis, err = quic.InheritedListen(laddr, tlsConfig, nil)

	} else {
		lis, err = inherit_net.Listen(network, laddr)
		if err == nil && tlsConfig != nil {
			if len(tlsConfig.Certificates) == 0 && tlsConfig.GetCertificate == nil {
				return nil, errors.New("tls: neither Certificates nor GetCertificate set in Config")
			}
			lis = tls.NewListener(lis, tlsConfig)
		}
	}

	if err == nil {
		pushParentLaddr(network, host, lis.Addr().String())
	}
	return
}

const parentLaddrsKey = "LISTEN_PARENT_ADDRS"

var parentAddrList = make(map[string]map[string][]string, 2) // network:host:[host:port]
var parentAddrListMutex sync.Mutex

func initParentLaddrList() {
	parentLaddr := os.Getenv(parentLaddrsKey)
	json.Unmarshal(goutil.StringToBytes(parentLaddr), &parentAddrList)
}

func setParentLaddrList() {
	b, _ := json.Marshal(parentAddrList)
	graceful.AddInherited(nil, []*graceful.Env{
		{K: parentLaddrsKey, V: goutil.BytesToString(b)},
	})
}

func pushParentLaddr(network, host, addr string) {
	parentAddrListMutex.Lock()
	defer parentAddrListMutex.Unlock()
	unifyLocalhost(&host)
	m, ok := parentAddrList[network]
	if !ok {
		m = make(map[string][]string)
		parentAddrList[network] = m
	}
	m[host] = append(m[host], addr)
}

func popParentLaddr(network, host, laddr string) string {
	parentAddrListMutex.Lock()
	defer parentAddrListMutex.Unlock()
	unifyLocalhost(&host)
	m, ok := parentAddrList[network]
	if !ok {
		return laddr
	}
	h, ok := m[host]
	if !ok {
		return laddr
	}
	if len(h) == 0 {
		return laddr
	}
	m[host] = h[1:]
	return h[0]
}

func unifyLocalhost(host *string) {
	switch *host {
	case "localhost":
		*host = "127.0.0.1"
	case "0.0.0.0":
		*host = "::"
	}
}
