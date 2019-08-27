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
	"strings"
	"sync"
	"time"

	"github.com/henrylee2cn/teleport/quic"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/coarsetime"
	"github.com/henrylee2cn/goutil/errors"
	"github.com/henrylee2cn/teleport/codec"
)

type (
	// BasePeer peer with the common method set
	BasePeer interface {
		// Close closes peer.
		Close() (err error)
		// CountSession returns the number of sessions.
		CountSession() int
		// GetSession gets the session by id.
		GetSession(sessionID string) (Session, bool)
		// RangeSession ranges all sessions. If fn returns false, stop traversing.
		RangeSession(fn func(sess Session) bool)
		// SetTLSConfig sets the TLS config.
		SetTLSConfig(tlsConfig *tls.Config)
		// SetTLSConfigFromFile sets the TLS config from file.
		SetTLSConfigFromFile(tlsCertFile, tlsKeyFile string, insecureSkipVerifyForClient ...bool) error
		// TLSConfig returns the TLS config.
		TLSConfig() *tls.Config
		// PluginContainer returns the global plugin container.
		PluginContainer() *PluginContainer
	}
	// EarlyPeer the communication peer that has just been created
	EarlyPeer interface {
		BasePeer
		// Router returns the root router of call or push handlers.
		Router() *Router
		// SubRoute adds handler group.
		SubRoute(pathPrefix string, plugin ...Plugin) *SubRouter
		// RouteCall registers CALL handlers, and returns the paths.
		RouteCall(ctrlStruct interface{}, plugin ...Plugin) []string
		// RouteCallFunc registers CALL handler, and returns the path.
		RouteCallFunc(callHandleFunc interface{}, plugin ...Plugin) string
		// RoutePush registers PUSH handlers, and returns the paths.
		RoutePush(ctrlStruct interface{}, plugin ...Plugin) []string
		// RoutePushFunc registers PUSH handler, and returns the path.
		RoutePushFunc(pushHandleFunc interface{}, plugin ...Plugin) string
		// SetUnknownCall sets the default handler, which is called when no handler for CALL is found.
		SetUnknownCall(fn func(UnknownCallCtx) (interface{}, *Status), plugin ...Plugin)
		// SetUnknownPush sets the default handler, which is called when no handler for PUSH is found.
		SetUnknownPush(fn func(UnknownPushCtx) *Status, plugin ...Plugin)
	}
	// Peer the communication peer which is server or client role
	Peer interface {
		EarlyPeer
		// ListenAndServe turns on the listening service.
		ListenAndServe(protoFunc ...ProtoFunc) error
		// Dial connects with the peer of the destination address.
		Dial(addr string, protoFunc ...ProtoFunc) (Session, *Status)
		// ServeConn serves the connection and returns a session.
		// NOTE:
		//  Not support automatically redials after disconnection;
		//  Execute the PostAcceptPlugin plugins.
		ServeConn(conn net.Conn, protoFunc ...ProtoFunc) (Session, *Status)
	}
)

var (
	_ BasePeer  = new(peer)
	_ EarlyPeer = new(peer)
	_ Peer      = new(peer)
)

type peer struct {
	router            *Router
	pluginContainer   *PluginContainer
	sessHub           *SessionHub
	closeCh           chan struct{}
	defaultSessionAge time.Duration // Default session max age, if less than or equal to 0, no time limit
	defaultContextAge time.Duration // Default CALL or PUSH context max age, if less than or equal to 0, no time limit
	tlsConfig         *tls.Config
	slowCometDuration time.Duration
	timeNow           func() int64
	mu                sync.Mutex
	network           string
	defaultBodyCodec  byte
	printDetail       bool
	countTime         bool

	// only for server role
	listenAddr string
	listeners  map[net.Listener]struct{}

	// only for client role
	defaultDialTimeout time.Duration
	redialInterval     time.Duration
	localAddr          net.Addr
	redialTimes        int32
}

// NewPeer creates a new peer.
func NewPeer(cfg PeerConfig, globalLeftPlugin ...Plugin) Peer {
	doPrintPid()
	pluginContainer := newPluginContainer()
	pluginContainer.AppendLeft(globalLeftPlugin...)
	pluginContainer.preNewPeer(&cfg)
	if err := cfg.check(); err != nil {
		Fatalf("%v", err)
	}

	var p = &peer{
		router:             newRouter(pluginContainer),
		pluginContainer:    pluginContainer,
		sessHub:            newSessionHub(),
		defaultSessionAge:  cfg.DefaultSessionAge,
		defaultContextAge:  cfg.DefaultContextAge,
		closeCh:            make(chan struct{}),
		slowCometDuration:  cfg.slowCometDuration,
		defaultDialTimeout: cfg.DefaultDialTimeout,
		redialInterval:     cfg.RedialInterval,
		network:            cfg.Network,
		listenAddr:         cfg.listenAddrStr,
		localAddr:          cfg.localAddr,
		printDetail:        cfg.PrintDetail,
		countTime:          cfg.CountTime,
		redialTimes:        cfg.RedialTimes,
		listeners:          make(map[net.Listener]struct{}),
	}

	if c, err := codec.GetByName(cfg.DefaultBodyCodec); err != nil {
		Fatalf("%v", err)
	} else {
		p.defaultBodyCodec = c.ID()
	}
	if p.countTime {
		p.timeNow = func() int64 { return time.Now().UnixNano() }
	} else {
		p.timeNow = func() int64 { return 0 }
	}
	addPeer(p)
	p.pluginContainer.postNewPeer(p)
	return p
}

// PluginContainer returns the global plugin container.
func (p *peer) PluginContainer() *PluginContainer {
	return p.pluginContainer
}

// TLSConfig returns the TLS config.
func (p *peer) TLSConfig() *tls.Config {
	return p.tlsConfig
}

// SetTLSConfig sets the TLS config.
func (p *peer) SetTLSConfig(tlsConfig *tls.Config) {
	p.tlsConfig = tlsConfig
}

// SetTLSConfigFromFile sets the TLS config from file.
func (p *peer) SetTLSConfigFromFile(tlsCertFile, tlsKeyFile string, insecureSkipVerifyForClient ...bool) error {
	var err error
	p.tlsConfig, err = NewTLSConfigFromFile(tlsCertFile, tlsKeyFile, insecureSkipVerifyForClient...)
	return err
}

// GetSession gets the session by id.
func (p *peer) GetSession(sessionID string) (Session, bool) {
	return p.sessHub.Get(sessionID)
}

// RangeSession ranges all sessions.
// If fn returns false, stop traversing.
func (p *peer) RangeSession(fn func(sess Session) bool) {
	p.sessHub.sessions.Range(func(key, value interface{}) bool {
		return fn(value.(*session))
	})
}

// CountSession returns the number of sessions.
func (p *peer) CountSession() int {
	return p.sessHub.sessions.Len()
}

// Dial connects with the peer of the destination address.
func (p *peer) Dial(addr string, protoFunc ...ProtoFunc) (Session, *Status) {
	return p.newSessionForClient(func() (net.Conn, error) {
		if p.network == "quic" {
			ctx := context.Background()
			if p.defaultDialTimeout > 0 {
				ctx, _ = context.WithTimeout(ctx, p.defaultDialTimeout)
			}
			if p.tlsConfig == nil {
				return quic.DialAddrContext(ctx, addr, &tls.Config{InsecureSkipVerify: true}, nil)
			}
			return quic.DialAddrContext(ctx, addr, p.tlsConfig, nil)
		}
		d := &net.Dialer{
			LocalAddr: p.localAddr,
			Timeout:   p.defaultDialTimeout,
		}
		if p.tlsConfig != nil {
			return tls.DialWithDialer(d, p.network, addr, p.tlsConfig)
		}
		return d.Dial(p.network, addr)
	}, addr, protoFunc)
}

type redialTimes int32

func (p *peer) newRedialTimes() *redialTimes {
	r := redialTimes(p.redialTimes)
	return &r
}

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

func (p *peer) newSessionForClient(dialFunc func() (net.Conn, error), addr string, protoFuncs []ProtoFunc) (*session, *Status) {
	conn, dialErr := dialFunc()
	if dialErr != nil {
		redialTimes := p.newRedialTimes()
		for redialTimes.next() {
			time.Sleep(p.redialInterval)
			Debugf("trying to redial... (network:%s, addr:%s)", p.network, addr)
			conn, dialErr = dialFunc()
			if dialErr == nil {
				break
			}
		}
	}
	if dialErr != nil {
		return nil, statDialFailed.Copy(dialErr)
	}
	var sess = newSession(p, conn, protoFuncs)

	// create redial func
	if p.redialTimes != 0 {
		sess.redialForClientLocked = func() bool {
			var stat *Status
			redialTimes := p.newRedialTimes()
			for redialTimes.next() {
				time.Sleep(p.redialInterval)
				Debugf("trying to redial... (network:%s, addr:%s, id:%s)", p.network, sess.RemoteAddr().String(), sess.ID())
				stat = p.renewSessionForClientLocked(sess, dialFunc, addr, protoFuncs)
				if stat.OK() {
					Infof("redial ok (network:%s, addr:%s, id:%s)", p.network, sess.RemoteAddr().String(), sess.ID())
					return true
				}
			}
			sess.tryChangeStatus(statusRedialFailed, statusRedialing)
			if !stat.OK() {
				Errorf("redial fail (network:%s, addr:%s, id:%s): %s", p.network, sess.RemoteAddr().String(), sess.ID(), stat.String())
			}
			return false
		}
	}

	sess.socket.SetID(sess.LocalAddr().String())
	if stat := p.pluginContainer.postDial(sess); !stat.OK() {
		sess.Close()
		return nil, stat
	}
	Infof("dial ok (network:%s, addr:%s, id:%s)", p.network, sess.RemoteAddr().String(), sess.ID())
	sess.changeStatus(statusOk)
	AnywayGo(sess.startReadAndHandle)
	p.sessHub.Set(sess)
	return sess, nil
}

func (p *peer) renewSessionForClientLocked(sess *session, dialFunc func() (net.Conn, error), addr string, protoFuncs []ProtoFunc) *Status {
	var conn, dialErr = dialFunc()
	if dialErr != nil {
		return statDialFailed.Copy(dialErr)
	}
	if p.tlsConfig != nil {
		conn = tls.Client(conn, p.tlsConfig)
	}
	oldIP := sess.LocalAddr().String()
	oldID := sess.ID()
	oldConn := sess.getConn()
	if oldConn != nil {
		oldConn.Close()
	}
	sess.socket.Reset(conn, protoFuncs...)
	if oldIP == oldID {
		sess.socket.SetID(sess.LocalAddr().String())
	} else {
		sess.socket.SetID(oldID)
	}
	sess.changeStatus(statusPreparing)
	if stat := p.pluginContainer.postDial(sess); !stat.OK() {
		sess.closeLocked()
		return stat
	}
	sess.changeStatus(statusOk)
	AnywayGo(sess.startReadAndHandle)
	p.sessHub.Set(sess)
	return nil
}

// ServeConn serves the connection and returns a session.
// NOTE:
//  Not support automatically redials after disconnection;
//  Execute the PostAcceptPlugin plugins.
func (p *peer) ServeConn(conn net.Conn, protoFunc ...ProtoFunc) (Session, *Status) {
	network := conn.LocalAddr().Network()
	if strings.Contains(network, "udp") {
		if _, ok := conn.(*quic.Conn); !ok {
			return nil, NewStatus(CodeWrongConn, "Not support UDP", "network must be one of the following: tcp, tcp4, tcp6, unix, unixpacket or quic")
		}
		network = "quic"
	}
	var sess = newSession(p, conn, protoFunc)
	if stat := p.pluginContainer.postAccept(sess); !stat.OK() {
		sess.Close()
		return nil, stat
	}
	Infof("serve ok (network:%s, addr:%s, id:%s)", network, sess.RemoteAddr().String(), sess.ID())
	sess.changeStatus(statusOk)
	AnywayGo(sess.startReadAndHandle)
	p.sessHub.Set(sess)
	return sess, nil
}

// ErrListenClosed listener is closed error.
var ErrListenClosed = errors.New("listener is closed")

// serveListener serves the listener.
// NOTE: The caller ensures that the listener supports graceful shutdown.
func (p *peer) serveListener(lis net.Listener, protoFunc ...ProtoFunc) error {
	defer lis.Close()
	p.listeners[lis] = struct{}{}

	network := lis.Addr().Network()
	if _, ok := lis.(*quic.Listener); ok {
		network = "quic"
	}
	addr := lis.Addr().String()
	Printf("listen and serve (network:%s, addr:%s)", network, addr)

	p.pluginContainer.postListen(lis.Addr())

	var (
		tempDelay time.Duration // how long to sleep on accept failure
		closeCh   = p.closeCh
	)
	for {
		conn, e := lis.Accept()
		if e != nil {
			select {
			case <-closeCh:
				return ErrListenClosed
			default:
			}
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}

				Tracef("accept error: %s; retrying in %v", e.Error(), tempDelay)

				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0
		AnywayGo(func() {
			if c, ok := conn.(*tls.Conn); ok {
				if p.defaultSessionAge > 0 {
					c.SetReadDeadline(coarsetime.CeilingTimeNow().Add(p.defaultSessionAge))
				}
				if p.defaultContextAge > 0 {
					c.SetReadDeadline(coarsetime.CeilingTimeNow().Add(p.defaultContextAge))
				}
				if err := c.Handshake(); err != nil {
					Errorf("TLS handshake error from %s: %s", c.RemoteAddr(), err.Error())
					return
				}
			}
			var sess = newSession(p, conn, protoFunc)
			if stat := p.pluginContainer.postAccept(sess); !stat.OK() {
				sess.Close()
				return
			}
			Infof("accept ok (network:%s, addr:%s, id:%s)", network, sess.RemoteAddr().String(), sess.ID())
			p.sessHub.Set(sess)
			sess.changeStatus(statusOk)
			sess.startReadAndHandle()
		})
	}
}

// ListenAndServe turns on the listening service.
func (p *peer) ListenAndServe(protoFunc ...ProtoFunc) error {
	if len(p.listenAddr) == 0 {
		Fatalf("listen address can not be empty")
	}
	lis, err := NewInheritedListener(p.network, p.listenAddr, p.tlsConfig)
	if err != nil {
		Fatalf("%v", err)
	}
	return p.serveListener(lis, protoFunc...)
}

// Close closes peer.
func (p *peer) Close() (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = errors.Errorf("panic:%v\n%s", p, goutil.PanicTrace(2))
		}
	}()
	close(p.closeCh)
	for lis := range p.listeners {
		if _, ok := lis.(*quic.Listener); !ok {
			lis.Close()
		}
	}
	deletePeer(p)
	var (
		count int
		errCh = make(chan error, 10)
	)
	p.sessHub.Range(func(sess *session) bool {
		count++
		if !Go(func() {
			errCh <- sess.Close()
		}) {
			errCh <- sess.Close()
		}
		return true
	})
	for i := 0; i < count; i++ {
		err = errors.Merge(err, <-errCh)
	}
	close(errCh)
	for lis := range p.listeners {
		if qlis, ok := lis.(*quic.Listener); ok {
			err = errors.Merge(err, qlis.Close())
		}
	}
	return err
}

var ctxPool = sync.Pool{
	New: func() interface{} {
		return newReadHandleCtx()
	},
}

func (p *peer) getContext(s *session, withWg bool) *handlerCtx {
	if withWg {
		// count get context
		s.graceCtxWaitGroup.Add(1)
	}
	ctx := ctxPool.Get().(*handlerCtx)
	ctx.clean()
	ctx.reInit(s)
	return ctx
}

func (p *peer) putContext(ctx *handlerCtx, withWg bool) {
	if withWg {
		// count get context
		ctx.sess.graceCtxWaitGroup.Done()
	}
	ctxPool.Put(ctx)
}

// Router returns the root router of call or push handlers.
func (p *peer) Router() *Router {
	return p.router
}

// SubRoute adds handler group.
func (p *peer) SubRoute(pathPrefix string, plugin ...Plugin) *SubRouter {
	return p.router.SubRoute(pathPrefix, plugin...)
}

// RouteCall registers CALL handlers, and returns the paths.
func (p *peer) RouteCall(callCtrlStruct interface{}, plugin ...Plugin) []string {
	return p.router.RouteCall(callCtrlStruct, plugin...)
}

// RouteCallFunc registers CALL handler, and returns the path.
func (p *peer) RouteCallFunc(callHandleFunc interface{}, plugin ...Plugin) string {
	return p.router.RouteCallFunc(callHandleFunc, plugin...)
}

// RoutePush registers PUSH handlers, and returns the paths.
func (p *peer) RoutePush(pushCtrlStruct interface{}, plugin ...Plugin) []string {
	return p.router.RoutePush(pushCtrlStruct, plugin...)
}

// RoutePushFunc registers PUSH handler, and returns the path.
func (p *peer) RoutePushFunc(pushHandleFunc interface{}, plugin ...Plugin) string {
	return p.router.RoutePushFunc(pushHandleFunc, plugin...)
}

// SetUnknownCall sets the default handler,
// which is called when no handler for CALL is found.
func (p *peer) SetUnknownCall(fn func(UnknownCallCtx) (interface{}, *Status), plugin ...Plugin) {
	p.router.SetUnknownCall(fn, plugin...)
}

// SetUnknownPush sets the default handler,
// which is called when no handler for PUSH is found.
func (p *peer) SetUnknownPush(fn func(UnknownPushCtx) *Status, plugin ...Plugin) {
	p.router.SetUnknownPush(fn, plugin...)
}

// maybe useful

func (p *peer) getCallHandler(uriPath string) (*Handler, bool) {
	return p.router.subRouter.getCall(uriPath)
}

func (p *peer) getPushHandler(uriPath string) (*Handler, bool) {
	return p.router.subRouter.getPush(uriPath)
}
