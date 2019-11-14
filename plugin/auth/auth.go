// Package auth is a plugin for verifying peer at the first time.
//
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
package auth

import (
	"fmt"
	"net"
	"sync/atomic"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/goutil"
)

// NewBearerPlugin creates a auth bearer plugin for client.
func NewBearerPlugin(fn Bearer, infoSetting ...erpc.MessageSetting) erpc.Plugin {
	return &authBearerPlugin{
		bearerFunc: fn,
		msgSetting: infoSetting,
	}
}

// NewCheckerPlugin creates a auth checker plugin for server.
func NewCheckerPlugin(fn Checker, retSetting ...erpc.MessageSetting) erpc.Plugin {
	return &authCheckerPlugin{
		checkerFunc: fn,
		msgSetting:  retSetting,
	}
}

type (
	// Bearer initiates an authorization request and handles the response.
	Bearer func(sess Session, fn SendOnce) *erpc.Status
	// SendOnce sends authorization request once.
	SendOnce func(info, retRecv interface{}) *erpc.Status

	// Checker checks the authorization request.
	Checker func(sess Session, fn RecvOnce) (ret interface{}, stat *erpc.Status)
	// RecvOnce receives authorization request once.
	RecvOnce func(infoRecv interface{}) *erpc.Status

	// Session auth session provides SetID, RemoteAddr and Swap methods in base session
	Session interface {
		// Peer returns the peer.
		Peer() erpc.Peer
		// SetID sets the session id.
		SetID(newID string)
		// LocalAddr returns the local network address.
		LocalAddr() net.Addr
		// RemoteAddr returns the remote network address.
		RemoteAddr() net.Addr
		// Swap returns custom data swap of the session(socket).
		Swap() goutil.Map
	}
)

type authBearerPlugin struct {
	bearerFunc Bearer
	msgSetting []erpc.MessageSetting
}

type authCheckerPlugin struct {
	checkerFunc Checker
	msgSetting  []erpc.MessageSetting
}

var (
	_ erpc.PostDialPlugin   = new(authBearerPlugin)
	_ erpc.PostAcceptPlugin = new(authCheckerPlugin)
)

func (a *authBearerPlugin) Name() string {
	return "auth-bearer"
}

func (a *authCheckerPlugin) Name() string {
	return "auth-checker"
}

// MultiSendErr the error of multiple call SendOnce function
var MultiSendErr = erpc.NewStatus(
	erpc.CodeWriteFailed,
	"auth-bearer plugin usage is incorrect",
	"multiple call SendOnce function",
)

// MultiRecvErr the error of multiple call RecvOnce function
var MultiRecvErr = erpc.NewStatus(
	erpc.CodeInternalServerError,
	"auth-checker plugin usage is incorrect",
	"multiple call RecvOnce function",
)

func (a *authBearerPlugin) PostDial(sess erpc.PreSession, _ bool) *erpc.Status {
	if a.bearerFunc == nil {
		return nil
	}
	var called int32
	return a.bearerFunc(sess, func(info, retRecv interface{}) *erpc.Status {
		if !atomic.CompareAndSwapInt32(&called, 0, 1) {
			return MultiSendErr
		}
		stat := sess.PreSend(erpc.TypeAuthCall, "", info, nil, a.msgSetting...)
		if !stat.OK() {
			return stat
		}
		retMsg := sess.PreReceive(func(header erpc.Header) interface{} {
			if header.Mtype() != erpc.TypeAuthReply {
				return nil
			}
			return retRecv
		})
		if !retMsg.StatusOK() {
			return retMsg.Status()
		}
		if retMsg.Mtype() != erpc.TypeAuthReply {
			return erpc.NewStatus(
				erpc.CodeUnauthorized,
				erpc.CodeText(erpc.CodeUnauthorized),
				fmt.Sprintf("auth message(1st) expect: AUTH_REPLY, but received: %s",
					erpc.TypeText(retMsg.Mtype())),
			)
		}
		return nil
	})
}

func (a *authCheckerPlugin) PostAccept(sess erpc.PreSession) *erpc.Status {
	if a.checkerFunc == nil {
		return nil
	}
	var called int32
	ret, stat := a.checkerFunc(sess, func(infoRecv interface{}) *erpc.Status {
		if !atomic.CompareAndSwapInt32(&called, 0, 1) {
			return MultiRecvErr
		}
		infoMsg := sess.PreReceive(func(header erpc.Header) interface{} {
			if header.Mtype() != erpc.TypeAuthCall {
				return nil
			}
			return infoRecv
		})
		if !infoMsg.StatusOK() {
			return infoMsg.Status()
		}
		if infoMsg.Mtype() != erpc.TypeAuthCall {
			return erpc.NewStatus(
				erpc.CodeUnauthorized,
				erpc.CodeText(erpc.CodeUnauthorized),
				fmt.Sprintf("auth message(1st) expect: AUTH_CALL, but received: %s",
					erpc.TypeText(infoMsg.Mtype())),
			)
		}
		return nil
	})
	if stat == MultiRecvErr {
		sess.PreSend(erpc.TypeAuthReply, "", nil, stat, a.msgSetting...)
		return stat
	}
	stat2 := sess.PreSend(erpc.TypeAuthReply, "", ret, stat, a.msgSetting...)
	if !stat2.OK() {
		return stat2
	}
	return stat
}
