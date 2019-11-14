// Copyright 2015-2019 HenryLee. All Rights Reserved.
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

package erpc

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/henrylee2cn/erpc/v6/quic"
)

// Dialer dial-up connection
type Dialer struct {
	network        string
	localAddr      net.Addr
	tlsConfig      *tls.Config
	dialTimeout    time.Duration
	redialInterval time.Duration
	redialTimes    int32
}

// NewDialer creates a dialer.
func NewDialer(localAddr net.Addr, tlsConfig *tls.Config,
	dialTimeout, redialInterval time.Duration, redialTimes int32,
) *Dialer {
	return &Dialer{
		network:        localAddr.Network(),
		localAddr:      localAddr,
		tlsConfig:      tlsConfig,
		dialTimeout:    dialTimeout,
		redialInterval: redialInterval,
		redialTimes:    redialTimes,
	}
}

// Network returns the network.
func (d *Dialer) Network() string {
	return d.network
}

// LocalAddr returns the local address.
func (d *Dialer) LocalAddr() net.Addr {
	return d.localAddr
}

// TLSConfig returns the TLS config.
func (d *Dialer) TLSConfig() *tls.Config {
	return d.tlsConfig
}

// DialTimeout returns the dial timeout.
func (d *Dialer) DialTimeout() time.Duration {
	return d.dialTimeout
}

// RedialInterval returns the redial interval.
func (d *Dialer) RedialInterval() time.Duration {
	return d.redialInterval
}

// RedialTimes returns the redial times.
func (d *Dialer) RedialTimes() int32 {
	return d.redialTimes
}

// Dial dials the connection, and try again if it fails.
func (d *Dialer) Dial(addr string) (net.Conn, error) {
	return d.dialWithRetry(addr, "")
}

// dialWithRetry dials the connection, and try again if it fails.
// NOTE:
//  sessID is not empty only when the disconnection is redialing
func (d *Dialer) dialWithRetry(addr, sessID string) (net.Conn, error) {
	conn, err := d.dialOne(addr)
	if err == nil {
		return conn, nil
	}
	redialTimes := d.newRedialCounter()
	for redialTimes.Next() {
		time.Sleep(d.redialInterval)
		if sessID == "" {
			Debugf("trying to redial... (network:%s, addr:%s)", d.network, addr)
		} else {
			Debugf("trying to redial... (network:%s, addr:%s, id:%s)", d.network, addr, sessID)
		}
		conn, err = d.dialOne(addr)
		if err == nil {
			return conn, nil
		}
	}
	return nil, err
}

// dialOne dials the connection once.
func (d *Dialer) dialOne(addr string) (net.Conn, error) {
	if asQUIC(d.network) {
		ctx := context.Background()
		if d.dialTimeout > 0 {
			ctx, _ = context.WithTimeout(ctx, d.dialTimeout)
		}
		if d.tlsConfig == nil {
			return quic.DialAddrContext(ctx, addr, GenerateTLSConfigForClient(), nil)
		}
		return quic.DialAddrContext(ctx, addr, d.tlsConfig, nil)
	}
	dialer := &net.Dialer{
		LocalAddr: d.localAddr,
		Timeout:   d.dialTimeout,
	}
	if d.tlsConfig != nil {
		return tls.DialWithDialer(dialer, d.network, addr, d.tlsConfig)
	}
	return dialer.Dial(d.network, addr)
}

// newRedialCounter creates a new redial counter.
func (d *Dialer) newRedialCounter() *redialCounter {
	r := redialCounter(d.redialTimes)
	return &r
}

// redialCounter redial counter
type redialCounter int32

// Next returns whether there are still more redial times.
func (r *redialCounter) Next() bool {
	t := *r
	if t == 0 {
		return false
	}
	if t > 0 {
		*r--
	}
	return true
}
