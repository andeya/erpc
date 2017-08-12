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

	"github.com/henrylee2cn/teleport/socket"
)

type Peer struct {
	id         string
	plugin     PluginContainer
	router     *Router
	sessionHub *SessionHub
}

func NewPeer(id string, router *Router, plugin PluginContainer) *Peer {
	var p = &Peer{
		id:         id,
		router:     router,
		plugin:     plugin,
		sessionHub: newSessionHub(),
	}
	return p
}

func (p *Peer) NewSession(conn net.Conn, id ...string) *Session {
	var session = &Session{
		peer:   p,
		socket: socket.Wrap(conn, id...),
	}
	p.sessionHub.Set(session.Id(), session)
	return session
}
