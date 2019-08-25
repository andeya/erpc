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
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/pool"
	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/socket"
	"github.com/henrylee2cn/teleport/utils"
)

// ServiceMethodMapper mapper service method from prefix, recvName and funcName.
// NOTE:
//  @prefix is optional;
//  @name is required.
type ServiceMethodMapper func(prefix, name string) (serviceMethod string)

// SetServiceMethodMapper customizes your own service method mapper.
func SetServiceMethodMapper(mapper ServiceMethodMapper) {
	globalServiceMethodMapper = mapper
}

// HTTPServiceMethodMapper like most RPC services service method mapper.
// Such as: user/get
// It is the default mapper.
// The mapping rule of struct(func) name to service methods:
//  `AaBb` -> `/aa_bb`
//  `ABcXYz` -> `/abc_xyz`
//  `Aa__Bb` -> `/aa_bb`
//  `aa__bb` -> `/aa_bb`
//  `ABC__XYZ` -> `/abc_xyz`
//  `Aa_Bb` -> `/aa/bb`
//  `aa_bb` -> `/aa/bb`
//  `ABC_XYZ` -> `/abc/xyz`
//
func HTTPServiceMethodMapper(prefix, name string) string {
	return path.Join("/", prefix, toServiceMethods(name, '/', true))
}

// RPCServiceMethodMapper like most RPC services service method mapper.
// Such as: User.Get
// The mapping rule of struct(func) name to service methods:
//  `AaBb` -> `AaBb`
//  `ABcXYz` -> `ABcXYz`
//  `Aa__Bb` -> `Aa_Bb`
//  `aa__bb` -> `aa_bb`
//  `ABC__XYZ` -> `ABC_XYZ`
//  `Aa_Bb` -> `Aa.Bb`
//  `aa_bb` -> `aa.bb`
//  `ABC_XYZ` -> `ABC.XYZ`
//
func RPCServiceMethodMapper(prefix, name string) string {
	p := prefix + "." + toServiceMethods(name, '.', false)
	return strings.Trim(p, ".")
}

// toServiceMethods maps struct(func) name to service methods.
func toServiceMethods(name string, sep rune, toSnake bool) string {
	var a = []rune{}
	var last rune
	for _, r := range name {
		if last == '_' {
			if r == '_' {
				last = '\x00'
				continue
			} else {
				a[len(a)-1] = sep
			}
		}
		if last == '\x00' && r == '_' {
			continue
		}
		a = append(a, r)
		last = r
	}
	name = string(a)
	if toSnake {
		name = goutil.SnakeString(name)
		name = strings.Replace(name, "__", "_", -1)
		name = strings.Replace(name, string(sep)+"_", string(sep), -1)
	}
	return name
}

const (
	// MetaRealIP real IP metadata key
	MetaRealIP = "X-Real-IP"
	// MetaAcceptBodyCodec the key of body codec that the sender wishes to accept
	MetaAcceptBodyCodec = "X-Accept-Body-Codec"
)

// GetAcceptBodyCodec gets the body codec that the sender wishes to accept.
// NOTE: If the specified codec is invalid, the receiver will ignore the mate data.
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
	return c, c != codec.NilCodecID
}

// WithNothing nothing to do.
//  func WithNothing() MessageSetting
var WithNothing = socket.WithNothing

// Socket is a generic stream-oriented network connection.
//
// Multiple goroutines may invoke methods on a Socket simultaneously.
type Socket = socket.Socket

// Proto pack/unpack protocol scheme of socket message.
type Proto = socket.Proto

// ProtoFunc function used to create a custom Proto interface.
type ProtoFunc = socket.ProtoFunc

// IOWithReadBuffer implements buffered I/O with buffered reader.
type IOWithReadBuffer = socket.IOWithReadBuffer

// Message a socket message data.
type Message = socket.Message

// NewBodyFunc creates a new body by header.
type NewBodyFunc = socket.NewBodyFunc

// Header message header interface
type Header = socket.Header

// Body message body interface
type Body = socket.Body

// MessageSetting is a pipe function type for setting message.
type MessageSetting = socket.MessageSetting

// WithStatus sets the message status.
// TYPE:
//  func WithStatus(stat *Status) MessageSetting
var WithStatus = socket.WithStatus

// WithRealIP sets the real IP to metadata.
func WithRealIP(ip string) MessageSetting {
	return socket.WithAddMeta(MetaRealIP, ip)
}

// WithAcceptBodyCodec sets the body codec that the sender wishes to accept.
// NOTE: If the specified codec is invalid, the receiver will ignore the mate data.
func WithAcceptBodyCodec(bodyCodec byte) MessageSetting {
	if bodyCodec == codec.NilCodecID {
		return WithNothing()
	}
	return socket.WithAddMeta(MetaAcceptBodyCodec, strconv.FormatUint(uint64(bodyCodec), 10))
}

// WithContext sets the message handling context.
//  func WithContext(ctx context.Context) MessageSetting
var WithContext = socket.WithContext

// withMtype sets the message type.
func withMtype(mtype byte) MessageSetting {
	return func(m Message) {
		m.SetMtype(mtype)
	}
}

// WithServiceMethod sets the message service method.
// SUGGEST: max len ≤ 255!
//  func WithServiceMethod(serviceMethod string) MessageSetting
var WithServiceMethod = socket.WithServiceMethod

// WithAddMeta adds 'key=value' metadata argument.
// Multiple values for the same key may be added.
// SUGGEST: urlencoded string max len ≤ 65535!
//  func WithAddMeta(key, value string) MessageSetting
var WithAddMeta = socket.WithAddMeta

// WithSetMeta sets 'key=value' metadata argument.
// SUGGEST: urlencoded string max len ≤ 65535!
//  func WithSetMeta(key, value string) MessageSetting
var WithSetMeta = socket.WithSetMeta

// WithBodyCodec sets the body codec.
//  func WithBodyCodec(bodyCodec byte) MessageSetting
var WithBodyCodec = socket.WithBodyCodec

// WithBody sets the body object.
//  func WithBody(body interface{}) MessageSetting
var WithBody = socket.WithBody

// WithNewBody resets the function of geting body.
//  NOTE: newBodyFunc is only for reading form connection.
//  func WithNewBody(newBodyFunc socket.NewBodyFunc) MessageSetting
var WithNewBody = socket.WithNewBody

// WithXferPipe sets transfer filter pipe.
// NOTE: Panic if the filterID is not registered.
// SUGGEST: The length can not be bigger than 255!
//  func WithXferPipe(filterID ...byte) MessageSetting
var WithXferPipe = socket.WithXferPipe

// GetMessage gets a Message form message pool.
// NOTE:
//  newBodyFunc is only for reading form connection;
//  settings are only for writing to connection.
//  func GetMessage(settings ...MessageSetting) Message
var GetMessage = socket.GetMessage

// PutMessage puts a Message to message pool.
//  func PutMessage(m Message)
var PutMessage = socket.PutMessage

var (
	_maxGoroutinesAmount      = (1024 * 1024 * 8) / 8 // max memory 8GB (8KB/goroutine)
	_maxGoroutineIdleDuration time.Duration
	_gopool                   = pool.NewGoPool(_maxGoroutinesAmount, _maxGoroutineIdleDuration)
)

// SetGopool set or reset go pool config.
// NOTE: Make sure to call it before calling NewPeer() and Go()
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
	_gopool.MustGo(fn)
}

// MustGo always try to use goroutine callbacks
// until execution is complete or the context is canceled.
func MustGo(fn func(), ctx ...context.Context) error {
	return _gopool.MustGo(fn, ctx...)
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
	output    Message
	result    interface{}
	stat      *Status
	inputMeta *utils.Args
}

// NewFakeCallCmd creates a fake CallCmd.
func NewFakeCallCmd(serviceMethod string, arg, result interface{}, stat *Status) CallCmd {
	return &fakeCallCmd{
		output: socket.NewMessage(
			withMtype(TypeCall),
			socket.WithServiceMethod(serviceMethod),
			socket.WithBody(arg),
		),
		result: result,
		stat:   stat,
	}
}

var closedChan = func() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}()

// TracePeer trace back the peer.
func (f *fakeCallCmd) TracePeer() (Peer, bool) {
	return nil, false
}

// TraceSession trace back the session.
func (f *fakeCallCmd) TraceSession() (Session, bool) {
	return nil, false
}

// Done returns the chan that indicates whether it has been completed.
func (f *fakeCallCmd) Done() <-chan struct{} {
	return closedChan
}

// Output returns writed message.
func (f *fakeCallCmd) Output() Message {
	return f.output
}

// Context carries a deadline, a cancelation signal, and other values across
// API boundaries.
func (f *fakeCallCmd) Context() context.Context {
	return f.output.Context()
}

// Reply returns the call reply.
func (f *fakeCallCmd) Reply() (interface{}, *Status) {
	return f.result, f.stat
}

// StatusOK returns the call status is OK or not.
func (f *fakeCallCmd) StatusOK() bool {
	return f.stat.OK()
}

// Status returns the call error.
func (f *fakeCallCmd) Status() *Status {
	return f.stat
}

// InputBodyCodec gets the body codec type of the input message.
func (f *fakeCallCmd) InputBodyCodec() byte {
	return codec.NilCodecID
}

// InputMeta returns the header metadata of input message.
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

// NewTLSConfigFromFile creates a new TLS config.
func NewTLSConfigFromFile(tlsCertFile, tlsKeyFile string, insecureSkipVerifyForClient ...bool) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(tlsCertFile, tlsKeyFile)
	if err != nil {
		return nil, err
	}
	return newTLSConfig(cert, insecureSkipVerifyForClient...), nil
}

// GenerateTLSConfigForClient setup a bare-bones(skip verify) TLS config for client.
func GenerateTLSConfigForClient() *tls.Config {
	return &tls.Config{InsecureSkipVerify: true}
}

// GenerateTLSConfigForServer setup a bare-bones TLS config for server.
func GenerateTLSConfigForServer() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return newTLSConfig(cert)
}

func newTLSConfig(cert tls.Certificate, insecureSkipVerifyForClient ...bool) *tls.Config {
	var insecureSkipVerify bool
	if len(insecureSkipVerifyForClient) > 0 {
		insecureSkipVerify = insecureSkipVerifyForClient[0]
	}
	return &tls.Config{
		InsecureSkipVerify:       insecureSkipVerify,
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
	}
}

// ListenerAddress a listener address plugin
type ListenerAddress struct {
	addr net.Addr
	host string
	port string
}

var _ PostListenPlugin = new(ListenerAddress)

// Addr returns the address object.
func (la *ListenerAddress) Addr() net.Addr {
	return la.addr
}

// Port returns the port.
func (la *ListenerAddress) Port() string {
	return la.port
}

// Host returns the host.
func (la *ListenerAddress) Host() string {
	return la.host
}

// String returns the address string.
func (la *ListenerAddress) String() string {
	return la.addr.String()
}

// Name returns plugin name.
func (la *ListenerAddress) Name() string {
	return "ListenerAddressPlugin"
}

// PostListen gets the listener address.
func (la *ListenerAddress) PostListen(addr net.Addr) (err error) {
	la.addr = addr
	la.host, la.port, err = net.SplitHostPort(addr.String())
	return
}
