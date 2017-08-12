// Copyright 2017 HenryLee. All Rights Reserved.
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
	"github.com/henrylee2cn/goutil"
)

// SessionHub sessions hub
type SessionHub struct {
	// key: session id (ip, name and so on)
	// value: *Session
	sessions goutil.Map
}

// newSessionHub creates a new sessions hub.
func newSessionHub() *SessionHub {
	chub := &SessionHub{
		sessions: goutil.AtomicMap(),
	}
	return chub
}

// Set sets a *Session.
func (sh *SessionHub) Set(id string, session *Session) {
	_session, loaded := sh.sessions.LoadOrStore(id, session)
	if !loaded {
		return
	}
	if oldSession := _session.(*Session); session != oldSession {
		oldSession.Close()
	}
}

// Get gets *Session by id.
// If second returned arg is false, mean the *Session is not found.
func (sh *SessionHub) Get(id string) (*Session, bool) {
	_session, ok := sh.sessions.Load(id)
	if !ok {
		return nil, false
	}
	return _session.(*Session), true
}

// Range calls f sequentially for each id and *Session present in the session hub.
// If f returns false, range stops the iteration.
func (sh *SessionHub) Range(f func(string, *Session) bool) {
	sh.sessions.Range(func(key, value interface{}) bool {
		return f(key.(string), value.(*Session))
	})
}

// Random gets a *Session randomly.
// If third returned arg is false, mean no *Session is exist.
func (sh *SessionHub) Random() (string, *Session, bool) {
	id, session, exist := sh.sessions.Random()
	if !exist {
		return "", nil, false
	}
	return id.(string), session.(*Session), true
}

// Len returns the length of the session hub.
// Note: the count implemented using sync.Map may be inaccurate.
func (sh *SessionHub) Len() int {
	return sh.sessions.Len()
}

// Delete deletes the *Session for a id.
func (sh *SessionHub) Delete(id string) {
	sh.sessions.Delete(id)
}
