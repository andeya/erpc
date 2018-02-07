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

	"github.com/henrylee2cn/goutil"
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

// NewFakePullCmd creates a fake PullCmd.
func NewFakePullCmd(p Peer, uri string, args, reply interface{}, rerr *Rerror) PullCmd {
	output := socket.NewPacket(
		socket.WithPtype(TypePull),
		socket.WithUri(uri),
		socket.WithBody(args),
	)
	cmd := &pullCmd{
		output: output,
		reply:  reply,
		public: goutil.RwMap(),
		rerr:   rerr,
	}

	if peerObj, ok := p.(*peer); ok {
		cmd.sess = newSession(peerObj, nil, nil)
	}
	return cmd
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
			// NextProtos: []string{"http/1.1", "h2"},
			PreferServerCipherSuites: true,
		}
	}
	return tlsConfig, nil
}

var printPidOnce sync.Once

func doPrintPid() {
	printPidOnce.Do(func() {
		Printf("The current process PID: %d", os.Getpid())
	})
}
