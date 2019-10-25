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
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/henrylee2cn/teleport/quic"
)

// Dialer dial-up connection
type Dialer struct {
	Network        string
	LocalAddr      net.Addr
	TLSConfig      *tls.Config
	DialTimeout    time.Duration
	RedialInterval time.Duration
	RedialTimes    int32
}

// Dial dials the connection, and try again if it fails.
func (d *Dialer) Dial(addr, sessID string) (net.Conn, *Status) {
	conn, err := d.DialConn(addr)
	if err == nil {
		return conn, nil
	}
	redialTimes := d.newRedialTimes()
	for redialTimes.next() {
		time.Sleep(d.RedialInterval)
		if sessID == "" {
			Debugf("trying to redial... (network:%s, addr:%s)", d.Network, addr)
		} else {
			Debugf("trying to redial... (network:%s, addr:%s, id:%s)", d.Network, addr, sessID)
		}
		conn, err = d.DialConn(addr)
		if err == nil {
			return conn, nil
		}
	}
	return nil, statDialFailed.Copy(err)
}

// DialConn dials the connection once.
func (d *Dialer) DialConn(addr string) (net.Conn, error) {
	if d.Network == "quic" {
		ctx := context.Background()
		if d.DialTimeout > 0 {
			ctx, _ = context.WithTimeout(ctx, d.DialTimeout)
		}
		if d.TLSConfig == nil {
			return quic.DialAddrContext(ctx, addr, GenerateTLSConfigForClient(), nil)
		}
		return quic.DialAddrContext(ctx, addr, d.TLSConfig, nil)
	}
	dialer := &net.Dialer{
		LocalAddr: d.LocalAddr,
		Timeout:   d.DialTimeout,
	}
	if d.TLSConfig != nil {
		return tls.DialWithDialer(dialer, d.Network, addr, d.TLSConfig)
	}
	return dialer.Dial(d.Network, addr)
}

func (d *Dialer) newRedialTimes() *redialTimes {
	r := redialTimes(d.RedialTimes)
	return &r
}

type redialTimes int32

func (r *redialTimes) next() bool {
	t := *r
	if t == 0 {
		return false
	}
	if t > 0 {
		*r--
	}
	return true
}
