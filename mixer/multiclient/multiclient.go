// Package multiclient is a higher throughput client connection pool when transferring large messages (such as downloading files).
//
// Copyright 2018 HenryLee. All Rights Reserved.
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
package multiclient

import (
	"time"

	"github.com/henrylee2cn/goutil/pool"
	tp "github.com/henrylee2cn/teleport"
)

// MultiClient client session which is has connection pool
type MultiClient struct {
	addr string
	peer tp.Peer
	pool *pool.Workshop
}

// New creates a client session which is has connection pool.
func New(peer tp.Peer, addr string, sessMaxQuota int, sessMaxIdleDuration time.Duration, protoFunc ...tp.ProtoFunc) *MultiClient {
	newWorkerFunc := func() (pool.Worker, error) {
		sess, rerr := peer.Dial(addr, protoFunc...)
		return sess, rerr.ToError()
	}
	return &MultiClient{
		addr: addr,
		peer: peer,
		pool: pool.NewWorkshop(sessMaxQuota, sessMaxIdleDuration, newWorkerFunc),
	}
}

// Addr returns the address.
func (c *MultiClient) Addr() string {
	return c.addr
}

// Peer returns the peer.
func (c *MultiClient) Peer() tp.Peer {
	return c.peer
}

// Close closes the session.
func (c *MultiClient) Close() {
	c.pool.Close()
}

// Stats returns the current session pool stats.
func (c *MultiClient) Stats() pool.WorkshopStats {
	return c.pool.Stats()
}

// AsyncCall sends a message and receives reply asynchronously.
// If the arg is []byte or *[]byte type, it can automatically fill in the body codec name.
func (c *MultiClient) AsyncCall(
	uri string,
	arg interface{},
	result interface{},
	callCmdChan chan<- tp.CallCmd,
	setting ...tp.MessageSetting,
) tp.CallCmd {
	_sess, err := c.pool.Hire()
	if err != nil {
		callCmd := tp.NewFakeCallCmd(uri, arg, result, tp.ToRerror(err))
		if callCmdChan != nil && cap(callCmdChan) == 0 {
			tp.Panicf("*MultiClient.AsyncCall(): callCmdChan channel is unbuffered")
		}
		callCmdChan <- callCmd
		return callCmd
	}
	sess := _sess.(tp.Session)
	defer c.pool.Fire(sess)
	return sess.AsyncCall(uri, arg, result, callCmdChan, setting...)
}

// Call sends a message and receives reply.
// Note:
// If the arg is []byte or *[]byte type, it can automatically fill in the body codec name;
// If the session is a client role and PeerConfig.RedialTimes>0, it is automatically re-called once after a failure.
func (c *MultiClient) Call(uri string, arg interface{}, result interface{}, setting ...tp.MessageSetting) tp.CallCmd {
	callCmd := c.AsyncCall(uri, arg, result, make(chan tp.CallCmd, 1), setting...)
	<-callCmd.Done()
	return callCmd
}

// Push sends a message, but do not receives reply.
// Note:
// If the arg is []byte or *[]byte type, it can automatically fill in the body codec name;
// If the session is a client role and PeerConfig.RedialTimes>0, it is automatically re-called once after a failure.
func (c *MultiClient) Push(uri string, arg interface{}, setting ...tp.MessageSetting) *tp.Rerror {
	_sess, err := c.pool.Hire()
	if err != nil {
		return tp.ToRerror(err)
	}
	sess := _sess.(tp.Session)
	defer c.pool.Fire(sess)
	return sess.Push(uri, arg, setting...)
}
