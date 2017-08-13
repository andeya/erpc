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
)

// Peer peer which is server or client.
type Peer struct {
	id          string
	plugin      PluginContainer
	apiMap      *ApiMap
	sessionHub  *SessionHub
	freeContext *ApiContext
	ctxLock     sync.Mutex
}

// NewPeer creates a new peer.
func NewPeer(id string, apiMap *ApiMap, plugin PluginContainer) *Peer {
	var p = &Peer{
		id:         id,
		apiMap:     apiMap,
		plugin:     plugin,
		sessionHub: newSessionHub(),
	}
	return p
}

// NewSession returns a session with connection.
func (p *Peer) NewSession(conn net.Conn, id ...string) *Session {
	var session = NewSession(p, conn, id...)
	p.sessionHub.Set(session.Id(), session)
	return session
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
