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
	"crypto/tls"
	"errors"
	"net"

	"github.com/henrylee2cn/erpc/v6/quic"
	"github.com/henrylee2cn/goutil/graceful/inherit_net"
)

var testTLSConfig = GenerateTLSConfigForServer()

// NewInheritedListener creates a inherited listener.
func NewInheritedListener(addr net.Addr, tlsConfig *tls.Config) (lis net.Listener, err error) {
	laddr := addr.String()
	network := addr.Network()
	var host, port string
	switch raddr := addr.(type) {
	case *FakeAddr:
		host, port = raddr.Host(), raddr.Port()
	default:
		host, port, err = net.SplitHostPort(laddr)
		if err != nil {
			return nil, err
		}
	}

	if port == "0" {
		laddr = popParentLaddr(network, host, laddr)
	}

	if asQUIC(network) {
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
