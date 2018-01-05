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
	"errors"
	"math"
	"time"

	"github.com/henrylee2cn/cfgo"

	"github.com/henrylee2cn/teleport/socket"
)

// PeerConfig peer config
// Note:
//  yaml tag is used for github.com/henrylee2cn/cfgo
//  ini tag is used for github.com/henrylee2cn/ini
type PeerConfig struct {
	TlsCertFile         string        `yaml:"tls_cert_file"          ini:"tls_cert_file"          comment:"TLS certificate file path"`
	TlsKeyFile          string        `yaml:"tls_key_file"           ini:"tls_key_file"           comment:"TLS key file path"`
	DefaultReadTimeout  time.Duration `yaml:"default_read_timeout"   ini:"default_read_timeout"   comment:"Default maximum duration for reading; ns,µs,ms,s,m,h"`
	DefaultWriteTimeout time.Duration `yaml:"default_write_timeout"  ini:"default_write_timeout"  comment:"Default maximum duration for writing; ns,µs,ms,s,m,h"`
	SlowCometDuration   time.Duration `yaml:"slow_comet_duration"    ini:"slow_comet_duration"    comment:"Slow operation alarm threshold; ns,µs,ms,s ..."`
	slowCometDuration   time.Duration
	DefaultBodyCodec    string        `yaml:"default_body_codec"     ini:"default_body_codec"     comment:"Default body codec type id"`
	PrintBody           bool          `yaml:"print_body"             ini:"print_body"             comment:"Is print body or not"`
	CountTime           bool          `yaml:"count_time"             ini:"count_time"             comment:"Is count cost time or not"`
	DefaultDialTimeout  time.Duration `yaml:"default_dial_timeout"   ini:"default_dial_timeout"   comment:"Default maximum duration for dialing; for client role; ns,µs,ms,s,m,h"`
	RedialTimes         int32         `yaml:"redial_times"           ini:"redial_times"           comment:"The maximum times of attempts to redial, after the connection has been unexpectedly broken; for client role"`
	Network             string        `yaml:"network"                ini:"network"                comment:"Network; tcp, tcp4, tcp6, unix or unixpacket"`
	ListenAddress       string        `yaml:"listen_address"         ini:"listen_address"         comment:"Listen address; for server role"`
}

var _ cfgo.Config = new(PeerConfig)

// Reload Bi-directionally synchronizes config between YAML file and memory.
func (p *PeerConfig) Reload(bind cfgo.BindFunc) error {
	err := bind()
	if err != nil {
		return err
	}
	return p.check()
}

func (p *PeerConfig) check() error {
	switch p.Network {
	default:
		return errors.New("Invalid network config, refer to the following: tcp, tcp4, tcp6, unix or unixpacket.")
	case "":
		p.Network = "tcp"
	case "tcp", "tcp4", "tcp6", "unix", "unixpacket":
	}
	p.slowCometDuration = math.MaxInt64
	if p.SlowCometDuration > 0 {
		p.slowCometDuration = p.SlowCometDuration
	}
	if len(p.DefaultBodyCodec) == 0 {
		p.DefaultBodyCodec = "json"
	}
	return nil
}

// SetDefaultBodyCodec set the default header codec.
// Note:
//  If the codec.Codec named 'codecId' is not registered, it will panic;
//  It is not safe to call it concurrently.
func SetGlobalBodyCodec(codecId byte) {
	socket.SetDefaultBodyCodec(codecId)
}
