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

// GetPacket gets a *Packet form packet stack.
// Note:
//  newBodyFunc is only for reading form connection;
//  settings are only for writing to connection.
//  func GetPacket(settings ...socket.PacketSetting) *socket.Packet
var GetPacket = socket.GetPacket

// PutPacket puts a *socket.Packet to packet stack.
//  func PutPacket(p *socket.Packet)
var PutPacket = socket.PutPacket

// DefaultProtoFunc gets the default builder of socket communication protocol
//  func DefaultProtoFunc() socket.ProtoFunc
var DefaultProtoFunc = socket.DefaultProtoFunc

// SetDefaultProtoFunc sets the default builder of socket communication protocol
//  func SetDefaultProtoFunc(protoFunc socket.ProtoFunc)
var SetDefaultProtoFunc = socket.SetDefaultProtoFunc

// GetReadLimit gets the packet size upper limit of reading.
//  PacketSizeLimit() uint32
var GetReadLimit = socket.PacketSizeLimit

// SetPacketSizeLimit sets max packet size.
// If maxSize<=0, set it to max uint32.
//  func SetPacketSizeLimit(maxPacketSize uint32)
var SetReadLimit = socket.SetPacketSizeLimit

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

// SetReadBuffer sets the size of the operating system's
// receive buffer associated with the *net.TCP connection.
// Note: Uses the default value, if bytes=1.
//  func SetTCPReadBuffer(bytes int)
var SetTCPReadBuffer = socket.SetTCPReadBuffer

// SetWriteBuffer sets the size of the operating system's
// transmit buffer associated with the *net.TCP connection.
// Note: Uses the default value, if bytes=1.
//  func SetTCPWriteBuffer(bytes int)
var SetTCPWriteBuffer = socket.SetTCPWriteBuffer

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

func init() {
	Printf("The current process PID: %d", os.Getpid())
}
