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

// PutPacket puts a *socket.Packet to packet stack.
//  func PutPacket(p *socket.Packet)
var PutPacket = socket.PutPacket

func init() {
	Printf("The current process PID: %d", os.Getpid())
}

var (
	_maxGoroutinesAmount      int
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
func Go(fn func()) {
	setGopoolOnce.Do(func() {
		if _gopool == nil {
			SetGopool(_maxGoroutinesAmount, _maxGoroutineIdleDuration)
		}
	})
	var err error
	for {
		if err = _gopool.Go(fn); err != nil {
			Warnf("%s", err.Error())
			time.Sleep(time.Millisecond * 20)
		} else {
			break
		}
	}
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
