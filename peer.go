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

package teleport

import (
	"crypto/tls"
	"math"
	"net"
	"sync"
	"time"

	"github.com/henrylee2cn/goutil/errors"
	"github.com/henrylee2cn/goutil/pool"
)

// Peer peer which is server or client.
type Peer struct {
	PullRouter        *Router
	PushRouter        *Router
	pluginContainer   PluginContainer
	sessionHub        *SessionHub
	closeCh           chan struct{}
	freeContext       *readHandleCtx
	ctxLock           sync.Mutex
	readTimeout       time.Duration // readdeadline for underlying net.Conn
	writeTimeout      time.Duration // writedeadline for underlying net.Conn
	tlsConfig         *tls.Config
	slowCometDuration time.Duration
	defaultCodec      string
	defaultGzipLevel  int32
	gopool            *pool.GoPool
	printBody         bool
	mu                sync.Mutex

	// for client role
	dialTimeout time.Duration

	// for server role
	listenAddrs []string
	listens     []net.Listener
}

// NewPeer creates a new peer.
func NewPeer(cfg *Config, plugin ...Plugin) *Peer {
	var slowCometDuration time.Duration = math.MaxInt64
	if cfg.SlowCometDuration > 0 {
		slowCometDuration = cfg.SlowCometDuration
	}
	var pluginContainer = newPluginContainer()
	if err := pluginContainer.Add(plugin...); err != nil {
		Fatalf("%v", err)
	}
	var p = &Peer{
		PullRouter:        newPullRouter(pluginContainer),
		PushRouter:        newPushRouter(pluginContainer),
		pluginContainer:   pluginContainer,
		sessionHub:        newSessionHub(),
		readTimeout:       cfg.ReadTimeout,
		writeTimeout:      cfg.WriteTimeout,
		closeCh:           make(chan struct{}),
		slowCometDuration: slowCometDuration,
		dialTimeout:       cfg.DialTimeout,
		listenAddrs:       cfg.ListenAddrs,
		defaultCodec:      cfg.DefaultCodec,
		defaultGzipLevel:  cfg.DefaultGzipLevel,
		printBody:         cfg.PrintBody,
		gopool:            pool.NewGoPool(cfg.MaxGoroutinesAmount, cfg.MaxGoroutineIdleDuration),
	}
	addPeer(p)
	return p
}

// GetSession gets the session by id.
func (p *Peer) GetSession(sessionId string) (*Session, bool) {
	return p.sessionHub.Get(sessionId)
}

// ServeConn serves the connection and returns a session.
func (p *Peer) ServeConn(conn net.Conn, id ...string) *Session {
	var session = newSession(p, conn, id...)
	p.sessionHub.Set(session)
	return session
}

// Dial connects with the peer of the destination address.
func (p *Peer) Dial(addr string, id ...string) (*Session, error) {
	var conn, err = net.DialTimeout("tcp", addr, p.dialTimeout)
	if err != nil {
		return nil, err
	}
	if p.tlsConfig != nil {
		conn = tls.Client(conn, p.tlsConfig)
	}
	var sess = newSession(p, conn, id...)
	if err = p.pluginContainer.PostDial(sess); err != nil {
		sess.Close()
		return nil, err
	}
	p.sessionHub.Set(sess)
	Printf("dial(addr: %s, id: %s) ok", addr, sess.Id())
	return sess, nil
}

// ErrListenClosed listener is closed error.
var ErrListenClosed = errors.New("listener is closed")

// Listen turns on the listening service.
func (p *Peer) Listen() error {
	var (
		wg    sync.WaitGroup
		count = len(p.listenAddrs)
		errCh = make(chan error, count)
	)
	wg.Add(count)
	for _, addr := range p.listenAddrs {
		go func(addr string) {
			defer wg.Done()
			errCh <- p.listen(addr)
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

func (p *Peer) listen(addr string) error {
	var lis, err = listen(addr, p.tlsConfig)
	if err != nil {
		Fatalf("%v", err)
	}
	p.listens = append(p.listens, lis)
	defer lis.Close()

	Printf("listen(addr: %s) ok", addr)

	var (
		tempDelay time.Duration // how long to sleep on accept failure
		closeCh   = p.closeCh
		sess      *Session
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

		sess = newSession(p, rw)
		if e = p.pluginContainer.PostAccept(sess); e != nil {
			sess.Close()
			Tracef("accept session(addr: %s, id: %s) error: %s", sess.RemoteIp(), sess.Id(), err.Error())
		} else {
			p.sessionHub.Set(sess)
			Tracef("accept session(addr: %s, id: %s) ok", sess.RemoteIp(), sess.Id())
		}
	}
}

// Close closes peer.
func (p *Peer) Close() error {
	close(p.closeCh)
	delete(peers.list, p)
	var errs []error
	p.sessionHub.Range(func(sess *Session) bool {
		errs = append(errs, sess.Close())
		return true
	})
	return errors.Merge(errs...)
}

func (p *Peer) getContext(s *Session) *readHandleCtx {
	p.ctxLock.Lock()
	{
		// count get context
		s.graceWaitGroup.Add(1)
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

func (p *Peer) putContext(ctx *readHandleCtx) {
	p.ctxLock.Lock()
	{
		// count get context
		ctx.session.graceWaitGroup.Done()
	}
	ctx.clean()
	ctx.next = p.freeContext
	p.freeContext = ctx
	p.ctxLock.Unlock()
}
