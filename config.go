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
	"errors"
	"math"
	"net"
	"strconv"
	"time"

	"github.com/henrylee2cn/cfgo"
	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/socket"
)

// PeerConfig peer config
// NOTE:
//  yaml tag is used for github.com/henrylee2cn/cfgo
//  ini tag is used for github.com/henrylee2cn/ini
type PeerConfig struct {
	Network            string        `yaml:"network"              ini:"network"              comment:"Network; tcp, tcp4, tcp6, unix, unixpacket or quic"`
	LocalIP            string        `yaml:"local_ip"             ini:"local_ip"             comment:"Local IP"`
	ListenPort         uint16        `yaml:"listen_port"          ini:"listen_port"          comment:"Listen port; for server role"`
	DefaultDialTimeout time.Duration `yaml:"default_dial_timeout" ini:"default_dial_timeout" comment:"Default maximum duration for dialing; for client role; ns,µs,ms,s,m,h"`
	RedialTimes        int32         `yaml:"redial_times"         ini:"redial_times"         comment:"The maximum times of attempts to redial, after the connection has been unexpectedly broken; Unlimited when <0; for client role"`
	RedialInterval     time.Duration `yaml:"redial_interval"      ini:"redial_interval"      comment:"Interval of redialing each time, default 100ms; for client role; ns,µs,ms,s,m,h"`
	DefaultBodyCodec   string        `yaml:"default_body_codec"   ini:"default_body_codec"   comment:"Default body codec type id"`
	DefaultSessionAge  time.Duration `yaml:"default_session_age"  ini:"default_session_age"  comment:"Default session max age, if less than or equal to 0, no time limit; ns,µs,ms,s,m,h"`
	DefaultContextAge  time.Duration `yaml:"default_context_age"  ini:"default_context_age"  comment:"Default CALL or PUSH context max age, if less than or equal to 0, no time limit; ns,µs,ms,s,m,h"`
	SlowCometDuration  time.Duration `yaml:"slow_comet_duration"  ini:"slow_comet_duration"  comment:"Slow operation alarm threshold; ns,µs,ms,s ..."`
	PrintDetail        bool          `yaml:"print_detail"         ini:"print_detail"         comment:"Is print body and metadata or not"`
	CountTime          bool          `yaml:"count_time"           ini:"count_time"           comment:"Is count cost time or not"`

	localAddr         net.Addr
	listenAddrStr     string
	slowCometDuration time.Duration
	checked           bool
}

var _ cfgo.Config = new(PeerConfig)

// ListenerAddr returns the listener address.
func (p *PeerConfig) ListenerAddr() string {
	p.check()
	return p.listenAddrStr
}

// Reload Bi-directionally synchronizes config between YAML file and memory.
func (p *PeerConfig) Reload(bind cfgo.BindFunc) error {
	err := bind()
	if err != nil {
		return err
	}
	p.checked = false
	return p.check()
}

func (p *PeerConfig) check() error {
	if p.checked {
		return nil
	}
	p.checked = true
	if len(p.LocalIP) == 0 {
		p.LocalIP = "0.0.0.0"
	}
	var err error
	switch p.Network {
	default:
		return errors.New("Invalid network config, refer to the following: tcp, tcp4, tcp6, unix, unixpacket or quic")
	case "":
		p.Network = "tcp"
		fallthrough
	case "tcp", "tcp4", "tcp6":
		p.localAddr, err = net.ResolveTCPAddr(p.Network, net.JoinHostPort(p.LocalIP, "0"))
	case "unix", "unixpacket":
		p.localAddr, err = net.ResolveUnixAddr(p.Network, net.JoinHostPort(p.LocalIP, "0"))
	case "quic":
		p.localAddr, err = net.ResolveUDPAddr("udp", net.JoinHostPort(p.LocalIP, "0"))
	}
	if err != nil {
		return err
	}
	p.listenAddrStr = net.JoinHostPort(p.LocalIP, strconv.FormatUint(uint64(p.ListenPort), 10))
	p.slowCometDuration = math.MaxInt64
	if p.SlowCometDuration > 0 {
		p.slowCometDuration = p.SlowCometDuration
	}
	if len(p.DefaultBodyCodec) == 0 {
		p.DefaultBodyCodec = DefaultBodyCodec().Name()
	}
	if p.RedialInterval <= 0 {
		p.RedialInterval = time.Millisecond * 100
	}
	return nil
}

var defaultBodyCodec codec.Codec = new(codec.JSONCodec)

// DefaultBodyCodec gets the default body codec.
func DefaultBodyCodec() codec.Codec {
	return defaultBodyCodec
}

// SetDefaultBodyCodec sets the default body codec.
func SetDefaultBodyCodec(codecID byte) error {
	c, err := codec.Get(codecID)
	if err != nil {
		return err
	}
	defaultBodyCodec = c
	return nil
}

// DefaultProtoFunc gets the default builder of socket communication protocol
//  func DefaultProtoFunc() tp.ProtoFunc
var DefaultProtoFunc = socket.DefaultProtoFunc

// SetDefaultProtoFunc sets the default builder of socket communication protocol
//  func SetDefaultProtoFunc(protoFunc tp.ProtoFunc)
var SetDefaultProtoFunc = socket.SetDefaultProtoFunc

// GetReadLimit gets the message size upper limit of reading.
//  GetReadLimit() uint32
var GetReadLimit = socket.MessageSizeLimit

// SetReadLimit sets max message size.
// If maxSize<=0, set it to max uint32.
//  func SetReadLimit(maxMessageSize uint32)
var SetReadLimit = socket.SetMessageSizeLimit

// SetSocketKeepAlive sets whether the operating system should send
// keepalive messages on the connection.
// NOTE: If have not called the function, the system defaults are used.
//  func SetSocketKeepAlive(keepalive bool)
var SetSocketKeepAlive = socket.SetKeepAlive

// SetSocketKeepAlivePeriod sets period between keep alives.
// NOTE: if d<0, don't change the value.
//  func SetSocketKeepAlivePeriod(d time.Duration)
var SetSocketKeepAlivePeriod = socket.SetKeepAlivePeriod

// SocketReadBuffer returns the size of the operating system's
// receive buffer associated with the connection.
// NOTE: if using the system default value, bytes=-1 and isDefault=true.
//  func SocketReadBuffer() (bytes int, isDefault bool)
var SocketReadBuffer = socket.ReadBuffer

// SetSocketReadBuffer sets the size of the operating system's
// receive buffer associated with the connection.
// NOTE: if bytes<0, don't change the value.
//  func SetSocketReadBuffer(bytes int)
var SetSocketReadBuffer = socket.SetReadBuffer

// SocketWriteBuffer returns the size of the operating system's
// transmit buffer associated with the connection.
// NOTE: if using the system default value, bytes=-1 and isDefault=true.
//  func SocketWriteBuffer() (bytes int, isDefault bool)
var SocketWriteBuffer = socket.WriteBuffer

// SetSocketWriteBuffer sets the size of the operating system's
// transmit buffer associated with the connection.
// NOTE: if bytes<0, don't change the value.
//  func SetSocketWriteBuffer(bytes int)
var SetSocketWriteBuffer = socket.SetWriteBuffer

// SetSocketNoDelay controls whether the operating system should delay
// packet transmission in hopes of sending fewer packets (Nagle's
// algorithm).  The default is true (no delay), meaning that data is
// sent as soon as possible after a Write.
//  func SetSocketNoDelay(noDelay bool)
var SetSocketNoDelay = socket.SetNoDelay
