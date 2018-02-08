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

package tp

import (
	"context"
	"crypto/tls"
	"os"
	"sync"
	"time"

	"github.com/henrylee2cn/goutil/pool"
	"github.com/henrylee2cn/teleport/socket"
	"github.com/henrylee2cn/teleport/utils"
)

// GetPacket gets a *Packet form packet stack.
// Note:
//  newBodyFunc is only for reading form connection;
//  settings are only for writing to connection.
//  func GetPacket(settings ...socket.PacketSetting) *socket.Packet
var GetPacket = socket.GetPacket

// PutPacket puts a *socket.Packet to packet stack.
//  func PutPacket(p *socket.Packet)
var PutPacket = socket.PutPacket

var (
	_maxGoroutinesAmount      = (1024 * 1024 * 8) / 8 // max memory 8GB (8KB/goroutine)
	_maxGoroutineIdleDuration time.Duration
	_gopool                   = pool.NewGoPool(_maxGoroutinesAmount, _maxGoroutineIdleDuration)
)

// SetGopool set or reset go pool config.
// Note: Make sure to call it before calling NewPeer() and Go()
func SetGopool(maxGoroutinesAmount int, maxGoroutineIdleDuration time.Duration) {
	_maxGoroutinesAmount, _maxGoroutineIdleDuration := maxGoroutinesAmount, maxGoroutineIdleDuration
	if _gopool != nil {
		_gopool.Stop()
	}
	_gopool = pool.NewGoPool(_maxGoroutinesAmount, _maxGoroutineIdleDuration)
}

// Go similar to go func, but return false if insufficient resources.
func Go(fn func()) bool {
	if err := _gopool.Go(fn); err != nil {
		Warnf("%s", err.Error())
		return false
	}
	return true
}

// AnywayGo similar to go func, but concurrent resources are limited.
func AnywayGo(fn func()) {
TRYGO:
	if !Go(fn) {
		time.Sleep(time.Second)
		goto TRYGO
	}
}

var printPidOnce sync.Once

func doPrintPid() {
	printPidOnce.Do(func() {
		Printf("The current process PID: %d", os.Getpid())
	})
}

type fakePullCmd struct {
	output    *socket.Packet
	reply     interface{}
	rerr      *Rerror
	inputMeta *utils.Args
}

// NewFakePullCmd creates a fake PullCmd.
func NewFakePullCmd(uri string, args, reply interface{}, rerr *Rerror) PullCmd {
	return &fakePullCmd{
		output: socket.NewPacket(
			socket.WithPtype(TypePull),
			socket.WithUri(uri),
			socket.WithBody(args),
		),
		reply:     reply,
		rerr:      rerr,
		inputMeta: utils.AcquireArgs(),
	}
}

// Output returns writed packet.
func (f *fakePullCmd) Output() *socket.Packet {
	return f.output
}

// Context carries a deadline, a cancelation signal, and other values across
// API boundaries.
func (f *fakePullCmd) Context() context.Context {
	return f.output.Context()
}

// Result returns the pull result.
func (f *fakePullCmd) Result() (interface{}, *Rerror) {
	return f.reply, f.rerr
}

// Rerror returns the pull error.
func (f *fakePullCmd) Rerror() *Rerror {
	return f.rerr
}

// InputMeta returns the header metadata of input packet.
func (f *fakePullCmd) InputMeta() *utils.Args {
	return f.inputMeta
}

// CostTime returns the pulled cost time.
// If PeerConfig.CountTime=false, always returns 0.
func (f *fakePullCmd) CostTime() time.Duration {
	return 0
}

// NewTlsConfigFromFile creates a new TLS config.
func NewTlsConfigFromFile(tlsCertFile, tlsKeyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(tlsCertFile, tlsKeyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates:             []tls.Certificate{cert},
		NextProtos:               []string{"http/1.1", "h2"},
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		},
	}, nil
}
