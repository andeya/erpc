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
	"crypto/tls"
	"os"
	"sync"
	"time"

	"github.com/henrylee2cn/goutil/pool"

	"github.com/henrylee2cn/teleport/socket"
)

// GetSenderPacket returns a packet for sending.
//  func GetSenderPacket(typ int32, uri string, body interface{}, setting ...socket.PacketSetting) *socket.Packet
var GetSenderPacket = socket.GetSenderPacket

// GetReceiverPacket returns a packet for sending.
//  func GetReceiverPacket(bodyGetting func(*socket.Header) interface{}) *socket.Packet
var GetReceiverPacket = socket.GetReceiverPacket

// DefaultProtocolFunc gets the default builder of socket communication protocol
//  func DefaultProtocolFunc() ProtocolFunc
var DefaultProtocolFunc = socket.DefaultProtocolFunc

// SetDefaultProtocolFunc sets the default builder of socket communication protocol
//  func SetDefaultProtocolFunc(protocolFunc ProtocolFunc)
var SetDefaultProtocolFunc = socket.SetDefaultProtocolFunc

// PutPacket puts a *socket.Packet to packet stack.
//  func PutPacket(p *socket.Packet)
var PutPacket = socket.PutPacket

// GetReadLimit gets the packet size upper limit of reading.
//  func GetReadLimit() int64
var GetReadLimit = socket.GetReadLimit

// GetPacketReadLimit sets max packet size.
// If maxSize<=0, set it to max int64.
//  func SetReadLimit(maxPacketSize int64)
var SetReadLimit = socket.SetReadLimit

func init() {
	Printf("The current process PID: %d", os.Getpid())
}

var (
	_maxGoroutinesAmount      = (1024 * 1024 * 8) / 8 // max memory 8GB (8KB/goroutine)
	_maxGoroutineIdleDuration time.Duration
	_gopool                   *pool.GoPool
	setGopoolOnce             sync.Once
)

// SetGopool set or reset go pool config.
// Note: Make sure to call it before calling NewPeer() and Go()
func SetGopool(maxGoroutinesAmount int, maxGoroutineIdleDuration time.Duration) {
	_maxGoroutinesAmount, _maxGoroutineIdleDuration := maxGoroutinesAmount, maxGoroutineIdleDuration
	_gopool = pool.NewGoPool(_maxGoroutinesAmount, _maxGoroutineIdleDuration)
}

// Go go func
func Go(fn func()) bool {
	setGopoolOnce.Do(func() {
		if _gopool == nil {
			SetGopool(_maxGoroutinesAmount, _maxGoroutineIdleDuration)
		}
	})
	if err := _gopool.Go(fn); err != nil {
		Warnf("%s", err.Error())
		return false
	}
	return true
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

func newTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	var tlsConfig *tls.Config
	if len(certFile) > 0 && len(keyFile) > 0 {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, err
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			// NextProtos:               []string{"http/1.1", "h2"},
			PreferServerCipherSuites: true,
		}
	}
	return tlsConfig, nil
}
