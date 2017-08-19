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
	"net"
	"sync"
	"time"
)

// Peer peer which is server or client.
type Peer struct {
	*ApiMap
	id              string
	pluginContainer PluginContainer
	sessionHub      *SessionHub
	freeContext     *ApiContext
	readTimeout     time.Duration // readdeadline for underlying net.Conn
	writeTimeout    time.Duration // writedeadline for underlying net.Conn
	closeCh         chan struct{}
	ctxLock         sync.Mutex
}

// NewPeer creates a new peer.
func NewPeer(id string, readTimeout, writeTimeout time.Duration) *Peer {
	var p = &Peer{
		id:              id,
		ApiMap:          newApiMap(),
		pluginContainer: newPluginContainer(),
		sessionHub:      newSessionHub(),
		readTimeout:     readTimeout,
		writeTimeout:    writeTimeout,
		closeCh:         make(chan struct{}),
	}
	return p
}

func (p *Peer) ServeConn(conn net.Conn, id ...string) {
	var session = newSession(p, conn, id...)
	p.sessionHub.Set(session.Id(), session)
}

func (p *Peer) getContext() *ApiContext {
	p.ctxLock.Lock()
	ctx := p.freeContext
	if ctx == nil {
		ctx = newApiContext()
	} else {
		p.freeContext = ctx.next
		*ctx = ApiContext{}
	}
	p.ctxLock.Unlock()
	return ctx
}

func (p *Peer) putContext(ctx *ApiContext) {
	p.ctxLock.Lock()
	ctx.clean()
	ctx.next = p.freeContext
	p.freeContext = ctx
	p.ctxLock.Unlock()
}

func (p *Peer) Close() {
	close(p.closeCh)
}
