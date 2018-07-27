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
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/pool"
	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/socket"
	"github.com/henrylee2cn/teleport/utils"
)

// Packet types
const (
	TypeUndefined byte = 0
	TypeCall      byte = 1
	TypeReply     byte = 2 // reply to call
	TypePush      byte = 3
)

// TypeText returns the packet type text.
// If the type is undefined returns 'Undefined'.
func TypeText(typ byte) string {
	switch typ {
	case TypeCall:
		return "CALL"
	case TypeReply:
		return "REPLY"
	case TypePush:
		return "PUSH"
	default:
		return "Undefined"
	}
}

// Internal Framework Rerror code.
// Note: Recommended custom code is greater than 1000.
//  unknown error code: -1.
//  sender peer error code range: [100,199].
//  packet handling error code range: [400,499].
//  receiver peer error code range: [500,599].
const (
	CodeUnknownError        = -1
	CodeConnClosed          = 102
	CodeWriteFailed         = 104
	CodeDialFailed          = 105
	CodeBadPacket           = 400
	CodeUnauthorized        = 401
	CodeNotFound            = 404
	CodePtypeNotAllowed     = 405
	CodeHandleTimeout       = 408
	CodeInternalServerError = 500
	CodeBadGateway          = 502

	// CodeConflict                      = 409
	// CodeUnsupportedTx                 = 410
	// CodeUnsupportedCodecType          = 415
	// CodeServiceUnavailable            = 503
	// CodeGatewayTimeout                = 504
	// CodeVariantAlsoNegotiates         = 506
	// CodeInsufficientStorage           = 507
	// CodeLoopDetected                  = 508
	// CodeNotExtended                   = 510
	// CodeNetworkAuthenticationRequired = 511
)

// CodeText returns the reply error code text.
// If the type is undefined returns 'Unknown Error'.
func CodeText(rerrCode int32) string {
	switch rerrCode {
	case CodeBadPacket:
		return "Bad Packet"
	case CodeUnauthorized:
		return "Unauthorized"
	case CodeDialFailed:
		return "Dial Failed"
	case CodeConnClosed:
		return "Connection Closed"
	case CodeWriteFailed:
		return "Write Failed"
	case CodeNotFound:
		return "Not Found"
	case CodeHandleTimeout:
		return "Handle Timeout"
	case CodePtypeNotAllowed:
		return "Packet Type Not Allowed"
	case CodeInternalServerError:
		return "Internal Server Error"
	case CodeBadGateway:
		return "Bad Gateway"
	case CodeUnknownError:
		fallthrough
	default:
		return "Unknown Error"
	}
}

// Internal Framework Rerror string.
var (
	rerrUnknownError        = NewRerror(CodeUnknownError, CodeText(CodeUnknownError), "")
	rerrDialFailed          = NewRerror(CodeDialFailed, CodeText(CodeDialFailed), "")
	rerrConnClosed          = NewRerror(CodeConnClosed, CodeText(CodeConnClosed), "")
	rerrWriteFailed         = NewRerror(CodeWriteFailed, CodeText(CodeWriteFailed), "")
	rerrBadPacket           = NewRerror(CodeBadPacket, CodeText(CodeBadPacket), "")
	rerrNotFound            = NewRerror(CodeNotFound, CodeText(CodeNotFound), "")
	rerrCodePtypeNotAllowed = NewRerror(CodePtypeNotAllowed, CodeText(CodePtypeNotAllowed), "")
	rerrHandleTimeout       = NewRerror(CodeHandleTimeout, CodeText(CodeHandleTimeout), "")
	rerrInternalServerError = NewRerror(CodeInternalServerError, CodeText(CodeInternalServerError), "")
)

// IsConnRerror determines whether the error is a connection error
func IsConnRerror(rerr *Rerror) bool {
	if rerr == nil {
		return false
	}
	if rerr.Code == CodeDialFailed || rerr.Code == CodeConnClosed {
		return true
	}
	return false
}

const (
	// MetaRerror reply error metadata key
	MetaRerror = "X-Reply-Error"
	// MetaRealIp real IP metadata key
	MetaRealIp = "X-Real-IP"
	// MetaAcceptBodyCodec the key of body codec that the sender wishes to accept
	MetaAcceptBodyCodec = "X-Accept-Body-Codec"
)

// WithRerror sets the real IP to metadata.
func WithRerror(rerr *Rerror) socket.PacketSetting {
	b, _ := rerr.MarshalJSON()
	if len(b) == 0 {
		return nil
	}
	return socket.WithAddMeta(MetaRerror, goutil.BytesToString(b))
}

// WithRealIp sets the real IP to metadata.
func WithRealIp(ip string) socket.PacketSetting {
	return socket.WithAddMeta(MetaRealIp, ip)
}

// WithAcceptBodyCodec sets the body codec that the sender wishes to accept.
// Note: If the specified codec is invalid, the receiver will ignore the mate data.
func WithAcceptBodyCodec(bodyCodec byte) socket.PacketSetting {
	if bodyCodec == codec.NilCodecId {
		return func(*socket.Packet) {}
	}
	return socket.WithAddMeta(MetaAcceptBodyCodec, strconv.FormatUint(uint64(bodyCodec), 10))
}

// GetAcceptBodyCodec gets the body codec that the sender wishes to accept.
// Note: If the specified codec is invalid, the receiver will ignore the mate data.
func GetAcceptBodyCodec(meta *utils.Args) (byte, bool) {
	s := meta.Peek(MetaAcceptBodyCodec)
	if len(s) == 0 || len(s) > 3 {
		return 0, false
	}
	b, err := strconv.ParseUint(goutil.BytesToString(s), 10, 8)
	if err != nil {
		return 0, false
	}
	c := byte(b)
	return c, c != codec.NilCodecId
}

// WithContext sets the packet handling context.
//  func WithContext(ctx context.Context) socket.PacketSetting
var WithContext = socket.WithContext

// WithSeq sets the packet sequence.
//  func WithSeq(seq uint64) socket.PacketSetting
var WithSeq = socket.WithSeq

// WithPtype sets the packet type.
//  func WithPtype(ptype byte) socket.PacketSetting
var WithPtype = socket.WithPtype

// WithUri sets the packet URI string.
//  func WithUri(uri string) socket.PacketSetting
var WithUri = socket.WithUri

// WithUriObject sets the packet URI object.
//  func WithUriObject(uriObject *url.URL) socket.PacketSetting
var WithUriObject = socket.WithUriObject

// WithQuery sets the packet URI query parameter.
//  func WithQuery(key, value string) socket.PacketSetting
var WithQuery = socket.WithQuery

// WithAddMeta adds 'key=value' metadata argument.
// Multiple values for the same key may be added.
//  func WithAddMeta(key, value string) socket.PacketSetting
var WithAddMeta = socket.WithAddMeta

// WithSetMeta sets 'key=value' metadata argument.
//  func WithSetMeta(key, value string) socket.PacketSetting
var WithSetMeta = socket.WithSetMeta

// WithBodyCodec sets the body codec.
//  func WithBodyCodec(bodyCodec byte) socket.PacketSetting
var WithBodyCodec = socket.WithBodyCodec

// WithBody sets the body object.
//  func WithBody(body interface{}) socket.PacketSetting
var WithBody = socket.WithBody

// WithNewBody resets the function of geting body.
//  func WithNewBody(newBodyFunc socket.NewBodyFunc) socket.PacketSetting
var WithNewBody = socket.WithNewBody

// WithXferPipe sets transfer filter pipe.
//  func WithXferPipe(filterId ...byte) socket.PacketSetting
// NOTE:
//  panic if the filterId is not registered
var WithXferPipe = socket.WithXferPipe

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

// TryGo tries to execute the function via goroutine.
// If there are no concurrent resources, execute it synchronously.
func TryGo(fn func()) {
	_gopool.TryGo(fn)
}

var printPidOnce sync.Once

func doPrintPid() {
	printPidOnce.Do(func() {
		Printf("The current process PID: %d", os.Getpid())
	})
}

type fakeCallCmd struct {
	output    *socket.Packet
	result    interface{}
	rerr      *Rerror
	inputMeta *utils.Args
}

// NewFakeCallCmd creates a fake CallCmd.
func NewFakeCallCmd(uri string, arg, result interface{}, rerr *Rerror) CallCmd {
	return &fakeCallCmd{
		output: socket.NewPacket(
			socket.WithPtype(TypeCall),
			socket.WithUri(uri),
			socket.WithBody(arg),
		),
		result: result,
		rerr:   rerr,
	}
}

var closedChan = func() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}()

// Done returns the chan that indicates whether it has been completed.
func (f *fakeCallCmd) Done() <-chan struct{} {
	return closedChan
}

// Output returns writed packet.
func (f *fakeCallCmd) Output() *socket.Packet {
	return f.output
}

// Context carries a deadline, a cancelation signal, and other values across
// API boundaries.
func (f *fakeCallCmd) Context() context.Context {
	return f.output.Context()
}

// Reply returns the call reply.
func (f *fakeCallCmd) Reply() (interface{}, *Rerror) {
	return f.result, f.rerr
}

// Rerror returns the call error.
func (f *fakeCallCmd) Rerror() *Rerror {
	return f.rerr
}

// InputBodyCodec gets the body codec type of the input packet.
func (f *fakeCallCmd) InputBodyCodec() byte {
	return codec.NilCodecId
}

// InputMeta returns the header metadata of input packet.
func (f *fakeCallCmd) InputMeta() *utils.Args {
	if f.inputMeta == nil {
		f.inputMeta = utils.AcquireArgs()
	}
	return f.inputMeta
}

// CostTime returns the called cost time.
// If PeerConfig.CountTime=false, always returns 0.
func (f *fakeCallCmd) CostTime() time.Duration {
	return 0
}

// NewTlsConfigFromFile creates a new TLS config.
func NewTlsConfigFromFile(tlsCertFile, tlsKeyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(tlsCertFile, tlsKeyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates:             []tls.Certificate{cert},
		NextProtos:               []string{"http/1.1", "h2"},
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		},
	}, nil
}
