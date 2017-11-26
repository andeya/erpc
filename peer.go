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
	"context"
	"crypto/tls"
	"math"
	"net"
	"sync"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/coarsetime"
	"github.com/henrylee2cn/goutil/errors"
	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/socket"
)

// Peer peer which is server or client.
type Peer struct {
	PullRouter          *Router
	PushRouter          *Router
	pluginContainer     PluginContainer
	sessHub             *SessionHub
	closeCh             chan struct{}
	freeContext         *readHandleCtx
	ctxLock             sync.Mutex
	defaultReadTimeout  int32 // time.Duration // readdeadline for underlying net.Conn
	defaultWriteTimeout int32 // time.Duration // writedeadline for underlying net.Conn
	tlsConfig           *tls.Config
	slowCometDuration   time.Duration
	defaultBodyCodec    byte
	printBody           bool
	countTime           bool
	timeNow             func() time.Time
	timeSince           func(time.Time) time.Duration
	mu                  sync.Mutex

	// for client role
	defaultDialTimeout time.Duration

	// for server role
	listenAddrs []string
	listens     []net.Listener
}

// NewPeer creates a new peer.
func NewPeer(cfg *PeerConfig, plugin ...Plugin) *Peer {
	var slowCometDuration time.Duration = math.MaxInt64
	if cfg.SlowCometDuration > 0 {
		slowCometDuration = cfg.SlowCometDuration
	}
	var pluginContainer = newPluginContainer()
	if err := pluginContainer.Add(plugin...); err != nil {
		Fatalf("%v", err)
	}
	var p = &Peer{
		PullRouter:          newPullRouter(pluginContainer),
		PushRouter:          newPushRouter(pluginContainer),
		pluginContainer:     pluginContainer,
		sessHub:             newSessionHub(),
		defaultReadTimeout:  int32(cfg.DefaultReadTimeout),
		defaultWriteTimeout: int32(cfg.DefaultWriteTimeout),
		closeCh:             make(chan struct{}),
		slowCometDuration:   slowCometDuration,
		defaultDialTimeout:  cfg.DefaultDialTimeout,
		listenAddrs:         cfg.ListenAddrs,
		printBody:           cfg.PrintBody,
		countTime:           cfg.CountTime,
	}
	if c, err := codec.GetByName(cfg.DefaultBodyCodec); err != nil {
		Fatalf("%v", err)
	} else {
		p.defaultBodyCodec = c.Id()
	}
	if p.countTime {
		p.timeNow = time.Now
		p.timeSince = time.Since
	} else {
		t0 := time.Time{}
		p.timeNow = func() time.Time { return t0 }
		p.timeSince = func(time.Time) time.Duration { return 0 }
	}

	addPeer(p)
	return p
}

// GetSession gets the session by id.
func (p *Peer) GetSession(sessionId string) (Session, bool) {
	return p.sessHub.Get(sessionId)
}

// RangeSession ranges all sessions.
// If fn returns false, stop traversing.
func (p *Peer) RangeSession(fn func(sess Session) bool) {
	p.sessHub.sessions.Range(func(key, value interface{}) bool {
		return fn(value.(*session))
	})
}

// CountSession returns the number of sessions.
func (p *Peer) CountSession() int {
	return p.sessHub.sessions.Len()
}

// ServeConn serves the connection and returns a session.
func (p *Peer) ServeConn(conn net.Conn, protoFunc ...socket.ProtoFunc) Session {
	var session = newSession(p, conn, protoFunc)
	p.sessHub.Set(session)
	return session
}

// Dial connects with the peer of the destination address.
func (p *Peer) Dial(addr string, protoFunc ...socket.ProtoFunc) (Session, *Rerror) {
	var conn, err = net.DialTimeout("tcp", addr, p.defaultDialTimeout)
	if err != nil {
		rerr := rerror_dialFailed.Copy()
		rerr.Detail = err.Error()
		return nil, rerr
	}
	if p.tlsConfig != nil {
		conn = tls.Client(conn, p.tlsConfig)
	}
	var sess = newSession(p, conn, protoFunc)
	sess.socket.SetId(sess.LocalIp())
	if rerr := p.pluginContainer.PostDial(sess); rerr != nil {
		sess.Close()
		return nil, rerr
	}
	Go(sess.startReadAndHandle)
	p.sessHub.Set(sess)
	Infof("dial ok (addr: %s, id: %s)", addr, sess.Id())
	return sess, nil
}

// DialContext connects with the peer of the destination address,
// using the provided context.
func (p *Peer) DialContext(ctx context.Context, addr string, protoFunc ...socket.ProtoFunc) (Session, *Rerror) {
	var d net.Dialer
	var conn, err = d.DialContext(ctx, "tcp", addr)
	if err != nil {
		rerr := rerror_dialFailed.Copy()
		rerr.Detail = err.Error()
		return nil, rerr
	}
	if p.tlsConfig != nil {
		conn = tls.Client(conn, p.tlsConfig)
	}
	var sess = newSession(p, conn, protoFunc)
	sess.socket.SetId(sess.LocalIp())
	if rerr := p.pluginContainer.PostDial(sess); rerr != nil {
		sess.Close()
		return nil, rerr
	}
	Go(sess.startReadAndHandle)
	p.sessHub.Set(sess)
	Infof("dial ok (addr: %s, id: %s)", addr, sess.Id())
	return sess, nil
}

// ErrListenClosed listener is closed error.
var ErrListenClosed = errors.New("listener is closed")

// Listen turns on the listening service.
func (p *Peer) Listen(protoFunc ...socket.ProtoFunc) error {
	var (
		wg    sync.WaitGroup
		count = len(p.listenAddrs)
		errCh = make(chan error, count)
	)
	wg.Add(count)
	for _, addr := range p.listenAddrs {
		go func(addr string) {
			defer wg.Done()
			errCh <- p.listen(addr, protoFunc)
		}(addr)
	}
	wg.Wait()
	close(errCh)
	var errs error
	for err := range errCh {
		e := err
		errs = errors.Append(errs, e)
	}
	return errs
}

func (p *Peer) listen(addr string, protoFuncs []socket.ProtoFunc) error {
	var lis, err = listen(addr, p.tlsConfig)
	if err != nil {
		Fatalf("%v", err)
	}
	p.listens = append(p.listens, lis)
	defer lis.Close()

	Printf("listen ok (addr: %s)", addr)

	var (
		tempDelay           time.Duration // how long to sleep on accept failure
		closeCh             = p.closeCh
		defaultReadTimeout  = time.Duration(p.defaultReadTimeout)
		defaultWriteTimeout = time.Duration(p.defaultWriteTimeout)
	)
	for {
		rw, e := lis.Accept()
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

		func(conn net.Conn) {
		TRYGO:
			if !Go(func() {
				if c, ok := conn.(*tls.Conn); ok {
					if defaultReadTimeout > 0 {
						c.SetReadDeadline(coarsetime.CeilingTimeNow().Add(defaultReadTimeout))
					}
					if defaultWriteTimeout > 0 {
						c.SetReadDeadline(coarsetime.CeilingTimeNow().Add(defaultWriteTimeout))
					}
					if err := c.Handshake(); err != nil {
						Errorf("TLS handshake error from %s: %s", c.RemoteAddr(), err.Error())
						return
					}
				}

				var sess = newSession(p, conn, protoFuncs)
				if err := p.pluginContainer.PostAccept(sess); err != nil {
					sess.Close()
					return
				}
				Tracef("accept session(addr: %s, id: %s) ok", sess.RemoteIp(), sess.Id())
				p.sessHub.Set(sess)
				sess.startReadAndHandle()
			}) {
				time.Sleep(time.Second)
				goto TRYGO
			}
		}(rw)
	}
}

// Close closes peer.
func (p *Peer) Close() (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = errors.Errorf("panic:\n%v\n%s", p, goutil.PanicTrace(2))
		}
	}()
	close(p.closeCh)
	delete(peers.list, p)
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
	return err
}

func (p *Peer) getContext(s *session, withWg bool) *readHandleCtx {
	p.ctxLock.Lock()
	if withWg {
		// count get context
		s.graceCtxWaitGroup.Add(1)
	}
	ctx := p.freeContext
	if ctx == nil {
		ctx = newReadHandleCtx()
	} else {
		p.freeContext = ctx.next
	}
	ctx.reInit(s)
	p.ctxLock.Unlock()
	return ctx
}

func (p *Peer) putContext(ctx *readHandleCtx, withWg bool) {
	p.ctxLock.Lock()
	defer p.ctxLock.Unlock()
	if withWg {
		// count get context
		ctx.sess.graceCtxWaitGroup.Done()
	}
	ctx.clean()
	ctx.next = p.freeContext
	p.freeContext = ctx
}
