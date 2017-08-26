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
	TlsCertFile          string        `yaml:"tls_cert_file"           ini:"tls_cert_file"           comment:"TLS certificate file path"`
	TlsKeyFile           string        `yaml:"tls_key_file"            ini:"tls_key_file"            comment:"TLS key file path"`
	DefaultReadTimeout   time.Duration `yaml:"default_read_timeout"    ini:"default_read_timeout"    comment:"Default maximum duration for reading; ns,µs,ms,s,m,h"`
	DefaultWriteTimeout  time.Duration `yaml:"default_write_timeout"   ini:"default_write_timeout"   comment:"Default maximum duration for writing; ns,µs,ms,s,m,h"`
	SlowCometDuration    time.Duration `yaml:"slow_comet_duration"     ini:"slow_comet_duration"     comment:"Slow operation alarm threshold; ns,µs,ms,s ..."`
	DefaultHeaderCodec   string        `yaml:"default_header_codec"    ini:"default_header_codec"    comment:"Default header codec"`
	DefaultBodyCodec     string        `yaml:"default_body_codec"      ini:"default_body_codec"      comment:"Default body codec"`
	DefaultBodyGzipLevel int32         `yaml:"default_body_gzip_level" ini:"default_body_gzip_level" comment:"Default body gzip level"`
	PrintBody            bool          `yaml:"print_body"              ini:"print_body"              comment:"Is print body or not"`
	DefaultDialTimeout   time.Duration `yaml:"default_dial_timeout"    ini:"default_dial_timeout"    comment:"Default maximum duration for dialing; for client role; ns,µs,ms,s,m,h"`
	ListenAddrs          []string      `yaml:"listen_addrs"            ini:"listen_addrs"            comment:"Listen addresses; for server role"`
}

var _ cfgo.Config = new(PeerConfig)

func (p *PeerConfig) Reload(bind cfgo.BindFunc) error {
	return bind()
}

// SetGlobalHeaderCodec sets global default header codec.
// Note:
//  If the codec.Codec named 'codecName' is not registered, it will panic;
//  It is not safe to call it concurrently.
func SetGlobalHeaderCodec(codecName string) {
	socket.SetDefaultHeaderCodec(codecName)
}

// SetGlobalBodyCodec sets global default body codec.
// Note:
//  If the codec.Codec named 'codecName' is not registered, it will panic;
//  It is not safe to call it concurrently.
func SetGlobalBodyCodec(codecName string) {
	socket.SetDefaultBodyCodec(codecName)
}
