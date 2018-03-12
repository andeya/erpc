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

package plugin

import (
	"fmt"

	"github.com/henrylee2cn/goutil"
	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/socket"
)

// A auth plugin for verifying peer at the first time.

// LaunchAuth creates a plugin for initiating authorization.
func LaunchAuth(fn GenerateAuthInfoFunc) tp.Plugin {
	return &auth{generateAuthInfoFunc: fn}
}

// VerifyAuth creates a plugin for verifying authorization.
func VerifyAuth(fn VerifyAuthInfoFunc) tp.Plugin {
	return &auth{verifyAuthInfoFunc: fn}
}

type (
	// AuthSession auth session provides Public,Id and SetId methods in early session
	AuthSession interface {
		Public() goutil.Map
		Id() string
		SetId(string)
	}
	// GenerateAuthInfoFunc the function used to generate auth info
	GenerateAuthInfoFunc func() string
	// VerifyAuthInfoFunc the function used to verify auth info
	VerifyAuthInfoFunc func(authInfo string, sess AuthSession) *tp.Rerror
	auth               struct {
		generateAuthInfoFunc GenerateAuthInfoFunc
		verifyAuthInfoFunc   VerifyAuthInfoFunc
	}
)

var (
	_ tp.PostDialPlugin   = new(auth)
	_ tp.PostAcceptPlugin = new(auth)
)

const authURI = "/auth/verify"

func (a *auth) Name() string {
	return "auth"
}

func (a *auth) PostDial(sess tp.EarlySession) *tp.Rerror {
	if a.generateAuthInfoFunc == nil {
		return nil
	}
	rerr := sess.Send(authURI, a.generateAuthInfoFunc(), nil, socket.WithBodyCodec('s'))
	if rerr != nil {
		return rerr
	}
	_, rerr = sess.Receive(func(header socket.Header) interface{} {
		return nil
	})
	return rerr
}

func (a *auth) PostAccept(sess tp.EarlySession) *tp.Rerror {
	if a.verifyAuthInfoFunc == nil {
		return nil
	}
	input, rerr := sess.Receive(func(header socket.Header) interface{} {
		if header.Uri() == authURI {
			return new(string)
		}
		return nil
	})
	if rerr != nil {
		return rerr
	}
	if input.Uri() != authURI {
		return tp.NewRerror(
			tp.CodeUnauthorized,
			tp.CodeText(tp.CodeUnauthorized),
			fmt.Sprintf("the 1th package want: PULL %s, but have: %s %s", authURI, tp.TypeText(input.Ptype()), input.Uri()),
		)
	}
	authInfo := *input.Body().(*string)
	rerr = a.verifyAuthInfoFunc(authInfo, sess)
	rerr2 := sess.Send(authURI, nil, rerr)
	if rerr == nil {
		rerr = rerr2
	}
	return rerr
}
