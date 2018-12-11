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

	"github.com/henrylee2cn/goutil"
	tp "github.com/henrylee2cn/teleport"
)

// A auth plugin for verifying peer at the first time.

// NewLaunchPlugin creates a plugin for initiating authorization.
func NewLaunchPlugin(fn GenerateAuthInfoFunc) tp.Plugin {
	return &auth{generateAuthInfoFunc: fn}
}

// NewVerifyPlugin creates a plugin for verifying authorization.
func NewVerifyPlugin(fn VerifyAuthInfoFunc) tp.Plugin {
	return &auth{verifyAuthInfoFunc: fn}
}

type (
	// Session auth session provides SetID, RemoteAddr and Swap methods in base session
	Session interface {
		// Peer returns the peer.
		Peer() tp.Peer
		// SetID sets the session id.
		SetID(newID string)
		// RemoteAddr returns the remote network address.
		RemoteAddr() net.Addr
		// Swap returns custom data swap of the session(socket).
		Swap() goutil.Map
	}
	// GenerateAuthInfoFunc the function used to generate auth info
	GenerateAuthInfoFunc func() string
	// VerifyAuthInfoFunc the function used to verify auth info
	VerifyAuthInfoFunc func(authInfo string, sess Session) *tp.Rerror
	auth               struct {
		generateAuthInfoFunc GenerateAuthInfoFunc
		verifyAuthInfoFunc   VerifyAuthInfoFunc
	}
)

var (
	_ tp.PostDialPlugin   = new(auth)
	_ tp.PostAcceptPlugin = new(auth)
)

const authServiceMethod = "/auth/verify"

func (a *auth) Name() string {
	return "auth"
}

func (a *auth) PostDial(sess tp.PreSession) *tp.Rerror {
	if a.generateAuthInfoFunc == nil {
		return nil
	}
	rerr := sess.Send(authServiceMethod, a.generateAuthInfoFunc(), nil, tp.WithBodyCodec('s'), tp.WithMtype(tp.TypeCall))
	if rerr != nil {
		return rerr
	}
	_, rerr = sess.Receive(func(header tp.Header) interface{} {
		return nil
	})
	return rerr
}

func (a *auth) PostAccept(sess tp.PreSession) *tp.Rerror {
	if a.verifyAuthInfoFunc == nil {
		return nil
	}
	input, rerr := sess.Receive(func(header tp.Header) interface{} {
		if header.Mtype() == tp.TypeCall && header.ServiceMethod() == authServiceMethod {
			return new(string)
		}
		return nil
	})
	if rerr != nil {
		return rerr
	}
	authInfoPtr, ok := input.Body().(*string)
	if !ok || input.Mtype() != tp.TypeCall || input.ServiceMethod() != authServiceMethod {
		rerr = tp.NewRerror(
			tp.CodeUnauthorized,
			tp.CodeText(tp.CodeUnauthorized),
			fmt.Sprintf("the 1th package want: CALL %s, but have: %s %s", authServiceMethod, tp.TypeText(input.Mtype()), input.ServiceMethod()),
		)
	} else {
		rerr = a.verifyAuthInfoFunc(*authInfoPtr, sess)
	}
	return sess.Send(authServiceMethod, nil, rerr, tp.WithMtype(tp.TypeReply))
}
