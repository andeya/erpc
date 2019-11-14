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
	"net"
	"sync"
	"time"

	"github.com/henrylee2cn/erpc/v6/codec"
	"github.com/henrylee2cn/erpc/v6/quic"
	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/coarsetime"
	"github.com/henrylee2cn/goutil/errors"
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
		//  Not check TLS;
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
	listenAddr net.Addr
	listeners  map[net.Listener]struct{}

	// only for client role
	dialer *Dialer
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
		router:            newRouter(pluginContainer),
		pluginContainer:   pluginContainer,
		sessHub:           newSessionHub(),
		defaultSessionAge: cfg.DefaultSessionAge,
		defaultContextAge: cfg.DefaultContextAge,
		closeCh:           make(chan struct{}),
		slowCometDuration: cfg.slowCometDuration,
		network:           cfg.Network,
		listenAddr:        cfg.listenAddr,
		printDetail:       cfg.PrintDetail,
		countTime:         cfg.CountTime,
		listeners:         make(map[net.Listener]struct{}),
		dialer: &Dialer{
			network:        cfg.Network,
			dialTimeout:    cfg.DialTimeout,
			localAddr:      cfg.localAddr,
			redialInterval: cfg.RedialInterval,
			redialTimes:    cfg.RedialTimes,
		},
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
	p.dialer.tlsConfig = tlsConfig
}

// SetTLSConfigFromFile sets the TLS config from file.
func (p *peer) SetTLSConfigFromFile(tlsCertFile, tlsKeyFile string, insecureSkipVerifyForClient ...bool) error {
	tlsConfig, err := NewTLSConfigFromFile(tlsCertFile, tlsKeyFile, insecureSkipVerifyForClient...)
	if err == nil {
		p.SetTLSConfig(tlsConfig)
	}
	return err
}

// GetSession gets the session by id.
func (p *peer) GetSession(sessionID string) (Session, bool) {
	return p.sessHub.get(sessionID)
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
	return p.sessHub.len()
}

// Dial connects with the peer of the destination address.
func (p *peer) Dial(addr string, protoFunc ...ProtoFunc) (Session, *Status) {
	conn, err := p.dialer.dialWithRetry(addr, "")
	if err != nil {
		return nil, statDialFailed.Copy(err)
	}
	sess := newSession(p, conn, protoFunc)

	// create redial func
	if p.dialer.RedialTimes() != 0 {
		sess.redialForClientLocked = func() bool {
			oldID := sess.ID()
			conn, err := p.dialer.dialWithRetry(addr, oldID)
			var stat *Status
			if err != nil {
				stat = statDialFailed.Copy(err)
			} else {
				oldIP := sess.LocalAddr().String()
				oldConn := sess.getConn()
				if oldConn != nil {
					oldConn.Close()
				}
				sess.socket.Reset(conn, protoFunc...)
				if oldIP == oldID {
					sess.socket.SetID(sess.LocalAddr().String())
				} else {
					sess.socket.SetID(oldID)
				}
				sess.changeStatus(statusPreparing)
				stat = p.pluginContainer.postDial(sess, true)
				if stat.OK() {
					sess.changeStatus(statusOk)
					AnywayGo(sess.startReadAndHandle)
					p.sessHub.set(sess)
					Infof("redial ok (network:%s, addr:%s, id:%s)", p.network, addr, sess.ID())
					return true
				}
				sess.closeLocked()
			}
			sess.tryChangeStatus(statusRedialFailed, statusRedialing)
			Errorf("redial fail (network:%s, addr:%s, id:%s): %s", p.network, addr, oldID, stat.String())
			return false
		}
	}

	sess.socket.SetID(sess.LocalAddr().String())
	if stat := p.pluginContainer.postDial(sess, false); !stat.OK() {
		sess.Close()
		return nil, stat
	}
	Infof("dial ok (network:%s, addr:%s, id:%s)", p.network, addr, sess.ID())
	sess.changeStatus(statusOk)
	AnywayGo(sess.startReadAndHandle)
	p.sessHub.set(sess)
	return sess, nil
}

// ServeConn serves the connection and returns a session.
// NOTE:
//  Not support automatically redials after disconnection;
//  Not check TLS;
//  Execute the PostAcceptPlugin plugins.
func (p *peer) ServeConn(conn net.Conn, protoFunc ...ProtoFunc) (Session, *Status) {
	network := conn.LocalAddr().Network()
	if asQUIC(network) {
		if _, ok := conn.(*quic.Conn); !ok {
			return nil, NewStatus(CodeWrongConn, "Not support "+network, "network must be one of the following: tcp, tcp4, tcp6, unix, unixpacket or quic")
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
	p.sessHub.set(sess)
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
			p.sessHub.set(sess)
			sess.changeStatus(statusOk)
			sess.startReadAndHandle()
		})
	}
}

// ListenAndServe turns on the listening service.
func (p *peer) ListenAndServe(protoFunc ...ProtoFunc) error {
	lis, err := NewInheritedListener(p.listenAddr, p.tlsConfig)
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
		errCh = make(chan error, 1024)
	)
	p.sessHub.rangeCallback(func(sess *session) bool {
		count++
		MustGo(func() {
			errCh <- sess.Close()
		})
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
