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
	"math/rand"
	"sync"

	"github.com/henrylee2cn/goutil"
)

// connHub connections hub
type connHub struct {
	// key: conn id (ip, name and so on)
	// value: Conn
	conns goutil.Map
	rwmu  sync.RWMutex
}

var ConnHub = newConnHub()

func newConnHub() *connHub {
	chub := &connHub{
		// TODO: use sync.Map in go1.9
		conns: goutil.NormalMap(102400),
	}
	return chub
}

// Set sets a Conn.
func (c *connHub) Set(id string, conn Conn) {
	_conn, loaded := c.conns.LoadOrStore(id, conn)
	if !loaded {
		return
	}
	if oldConn := _conn.(Conn); conn != oldConn {
		oldConn.Close()
	}
}

// Get gets Conn by id.
// If second returned arg is false, mean the Conn is not found.
func (c *connHub) Get(id string) (Conn, bool) {
	_conn, ok := c.conns.Load(id)
	if !ok {
		return nil, false
	}
	return _conn.(Conn), true
}

// Range calls f sequentially for each id and Conn present in the conn hub.
// If f returns false, range stops the iteration.
func (c *connHub) Range(f func(string, Conn) bool) {
	c.conns.Range(func(key, value interface{}) bool {
		return f(key.(string), value.(Conn))
	})
}

// Random gets a Conn randomly.
// If third returned arg is false, mean no Conn is exist.
func (c *connHub) Random() (string, Conn, bool) {
	var id string
	var conn Conn
	c.rwmu.RLock()
	var length = c.conns.Len()
	if length == 0 {
		c.rwmu.RUnlock()
		return id, nil, false
	}
	var i = rand.Intn(length)
	c.conns.Range(func(key, value interface{}) bool {
		if i == 0 {
			id, conn = key.(string), value.(Conn)
			return false
		}
		i--
		return true
	})
	c.rwmu.RUnlock()
	return id, conn, conn != nil
}

// Len returns the length of the conn hub.
func (c *connHub) Len() int {
	return c.conns.Len()
}

// Delete deletes the Conn for a id.
func (c *connHub) Delete(id string) {
	c.rwmu.Lock()
	c.conns.Delete(id)
	c.rwmu.Unlock()
}
