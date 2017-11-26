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
	"time"

	"github.com/henrylee2cn/cfgo"

	"github.com/henrylee2cn/teleport/socket"
)

// PeerConfig peer config
// Note:
//  yaml tag is used for github.com/henrylee2cn/cfgo
//  ini tag is used for github.com/henrylee2cn/ini
type PeerConfig struct {
	TlsCertFile         string        `yaml:"tls_cert_file"           ini:"tls_cert_file"           comment:"TLS certificate file path"`
	TlsKeyFile          string        `yaml:"tls_key_file"            ini:"tls_key_file"            comment:"TLS key file path"`
	DefaultReadTimeout  time.Duration `yaml:"default_read_timeout"    ini:"default_read_timeout"    comment:"Default maximum duration for reading; ns,µs,ms,s,m,h"`
	DefaultWriteTimeout time.Duration `yaml:"default_write_timeout"   ini:"default_write_timeout"   comment:"Default maximum duration for writing; ns,µs,ms,s,m,h"`
	SlowCometDuration   time.Duration `yaml:"slow_comet_duration"     ini:"slow_comet_duration"     comment:"Slow operation alarm threshold; ns,µs,ms,s ..."`
	DefaultBodyCodec    string        `yaml:"default_body_codec"      ini:"default_body_codec"      comment:"Default body codec type id"`
	PrintBody           bool          `yaml:"print_body"              ini:"print_body"              comment:"Is print body or not"`
	CountTime           bool          `yaml:"count_time"              ini:"count_time"              comment:"Is count cost time or not"`
	DefaultDialTimeout  time.Duration `yaml:"default_dial_timeout"    ini:"default_dial_timeout"    comment:"Default maximum duration for dialing; for client role; ns,µs,ms,s,m,h"`
	ListenAddrs         []string      `yaml:"listen_addrs"            ini:"listen_addrs"            comment:"Listen addresses; for server role"`
}

var _ cfgo.Config = new(PeerConfig)

func (p *PeerConfig) Reload(bind cfgo.BindFunc) error {
	return bind()
}

// SetDefaultBodyCodec set the default header codec.
// Note:
//  If the codec.Codec named 'codecId' is not registered, it will panic;
//  It is not safe to call it concurrently.
func SetGlobalBodyCodec(codecId byte) {
	socket.SetDefaultBodyCodec(codecId)
}
