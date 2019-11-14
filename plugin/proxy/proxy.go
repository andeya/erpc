// Package proxy is a plugin for handling unknown calling or pushing.
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

package proxy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/goutil"
)

// NewPlugin creates a proxy plugin for handling unknown calling and pushing.
func NewPlugin(fn func(*Label) Forwarder) erpc.Plugin {
	return &proxy{
		callForwarder: func(label *Label) CallForwarder {
			return fn(label)
		},
		pushForwarder: func(label *Label) PushForwarder {
			return fn(label)
		},
	}
}

// NewCallPlugin creates a proxy plugin for handling unknown calling.
func NewCallPlugin(fn func(*Label) CallForwarder) erpc.Plugin {
	return &proxy{callForwarder: fn}
}

// NewPushPlugin creates a proxy plugin for handling unknown pushing.
func NewPushPlugin(fn func(*Label) PushForwarder) erpc.Plugin {
	return &proxy{pushForwarder: fn}
}

type (
	// Forwarder the object used to call and push
	Forwarder interface {
		CallForwarder
		PushForwarder
	}
	// CallForwarder the object used to call
	CallForwarder interface {
		Call(uri string, arg interface{}, result interface{}, setting ...erpc.MessageSetting) erpc.CallCmd
	}
	// PushForwarder the object used to push
	PushForwarder interface {
		Push(uri string, arg interface{}, setting ...erpc.MessageSetting) *erpc.Status
	}
	// Label proxy label information
	Label struct {
		SessionID, RealIP, ServiceMethod string
	}
	proxy struct {
		callForwarder func(*Label) CallForwarder
		pushForwarder func(*Label) PushForwarder
	}
)

var (
	_ erpc.PostNewPeerPlugin = new(proxy)
)

func (p *proxy) Name() string {
	return "proxy"
}

func (p *proxy) PostNewPeer(peer erpc.EarlyPeer) error {
	if p.callForwarder != nil {
		peer.SetUnknownCall(p.call)
	}
	if p.pushForwarder != nil {
		peer.SetUnknownPush(p.push)
	}
	return nil
}

func (p *proxy) call(ctx erpc.UnknownCallCtx) (interface{}, *erpc.Status) {
	var (
		label    Label
		settings = make([]erpc.MessageSetting, 0, 16)
	)
	label.SessionID = ctx.Session().ID()
	ctx.VisitMeta(func(key, value []byte) {
		settings = append(settings, erpc.WithAddMeta(string(key), string(value)))
	})
	var (
		result      []byte
		realIPBytes = ctx.PeekMeta(erpc.MetaRealIP)
	)
	if len(realIPBytes) == 0 {
		label.RealIP = ctx.IP()
		settings = append(settings, erpc.WithAddMeta(erpc.MetaRealIP, label.RealIP))
	} else {
		label.RealIP = goutil.BytesToString(realIPBytes)
	}
	label.ServiceMethod = ctx.ServiceMethod()
	callcmd := p.callForwarder(&label).Call(label.ServiceMethod, ctx.InputBodyBytes(), &result, settings...)
	callcmd.InputMeta().VisitAll(func(key, value []byte) {
		ctx.SetMeta(goutil.BytesToString(key), goutil.BytesToString(value))
	})
	stat := callcmd.Status()
	if !stat.OK() && stat.Code() < 200 && stat.Code() > 99 {
		stat.SetCode(erpc.CodeBadGateway)
		stat.SetMsg(erpc.CodeText(erpc.CodeBadGateway))
	}
	return result, stat
}

func (p *proxy) push(ctx erpc.UnknownPushCtx) *erpc.Status {
	var (
		label    Label
		settings = make([]erpc.MessageSetting, 0, 16)
	)
	label.SessionID = ctx.Session().ID()
	ctx.VisitMeta(func(key, value []byte) {
		settings = append(settings, erpc.WithAddMeta(string(key), string(value)))
	})
	if realIPBytes := ctx.PeekMeta(erpc.MetaRealIP); len(realIPBytes) == 0 {
		label.RealIP = ctx.IP()
		settings = append(settings, erpc.WithAddMeta(erpc.MetaRealIP, label.RealIP))
	} else {
		label.RealIP = goutil.BytesToString(realIPBytes)
	}
	label.ServiceMethod = ctx.ServiceMethod()
	stat := p.pushForwarder(&label).Push(label.ServiceMethod, ctx.InputBodyBytes(), settings...)
	if !stat.OK() && stat.Code() < 200 && stat.Code() > 99 {
		stat.SetCode(erpc.CodeBadGateway)
		stat.SetMsg(erpc.CodeText(erpc.CodeBadGateway))
	}
	return stat
}

var peerName = filepath.Base(os.Args[0])
var incr int64
var mutex sync.Mutex

// getSeq creates a new sequence with some prefix string.
func getSeq(prefix ...string) string {
	mutex.Lock()
	seq := fmt.Sprintf("%s[%d]", peerName, incr)
	incr++
	mutex.Unlock()
	for _, p := range prefix {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		seq = p + ">" + seq
	}
	return seq
}
