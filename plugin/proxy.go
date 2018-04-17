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
	"github.com/henrylee2cn/goutil"
	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/socket"
)

// A proxy plugin for handling unknown pulling or pushing.

// Proxy creates a proxy plugin for handling unknown pulling and pushing.
func Proxy(fn func(*ProxyLabel) Caller) tp.Plugin {
	return &proxy{
		pullCaller: func(label *ProxyLabel) PullCaller {
			return fn(label)
		},
		pushCaller: func(label *ProxyLabel) PushCaller {
			return fn(label)
		},
	}
}

// ProxyPull creates a proxy plugin for handling unknown pulling.
func ProxyPull(fn func(*ProxyLabel) PullCaller) tp.Plugin {
	return &proxy{pullCaller: fn}
}

// ProxyPush creates a proxy plugin for handling unknown pushing.
func ProxyPush(fn func(*ProxyLabel) PushCaller) tp.Plugin {
	return &proxy{pushCaller: fn}
}

type (
	// Caller the object used to pull and push
	Caller interface {
		PullCaller
		PushCaller
	}
	// PullCaller the object used to pull
	PullCaller interface {
		Pull(uri string, args interface{}, reply interface{}, setting ...socket.PacketSetting) tp.PullCmd
	}
	// PushCaller the object used to push
	PushCaller interface {
		Push(uri string, args interface{}, setting ...socket.PacketSetting) *tp.Rerror
	}
	// ProxyLabel proxy label information
	ProxyLabel struct {
		SessionId, RealIp, Uri string
	}
	proxy struct {
		pullCaller func(*ProxyLabel) PullCaller
		pushCaller func(*ProxyLabel) PushCaller
	}
)

var (
	_ tp.PostNewPeerPlugin = new(proxy)
)

func (p *proxy) Name() string {
	return "proxy"
}

func (p *proxy) PostNewPeer(peer tp.EarlyPeer) error {
	if p.pullCaller != nil {
		peer.SetUnknownPull(p.pull)
	}
	if p.pushCaller != nil {
		peer.SetUnknownPush(p.push)
	}
	return nil
}

func (p *proxy) pull(ctx tp.UnknownPullCtx) (interface{}, *tp.Rerror) {
	var (
		label    ProxyLabel
		settings = make([]socket.PacketSetting, 1, 8)
	)
	label.SessionId = ctx.Session().Id()
	settings[0] = tp.WithSeq(label.SessionId + "@" + ctx.Seq())
	ctx.VisitMeta(func(key, value []byte) {
		settings = append(settings, tp.WithAddMeta(string(key), string(value)))
	})
	var (
		reply       []byte
		realIpBytes = ctx.PeekMeta(tp.MetaRealIp)
	)
	if len(realIpBytes) == 0 {
		label.RealIp = ctx.Ip()
		settings = append(settings, tp.WithAddMeta(tp.MetaRealIp, label.RealIp))
	} else {
		label.RealIp = goutil.BytesToString(realIpBytes)
	}
	label.Uri = ctx.Uri()
	pullcmd := p.pullCaller(&label).Pull(label.Uri, ctx.InputBodyBytes(), &reply, settings...)
	pullcmd.InputMeta().VisitAll(func(key, value []byte) {
		ctx.SetMeta(goutil.BytesToString(key), goutil.BytesToString(value))
	})
	rerr := pullcmd.Rerror()
	if rerr != nil && rerr.Code < 200 && rerr.Code > 99 {
		rerr.Code = tp.CodeBadGateway
		rerr.Message = tp.CodeText(tp.CodeBadGateway)
	}
	return reply, rerr
}

func (p *proxy) push(ctx tp.UnknownPushCtx) *tp.Rerror {
	var (
		label    ProxyLabel
		settings = make([]socket.PacketSetting, 1, 8)
	)
	label.SessionId = ctx.Session().Id()
	settings[0] = tp.WithSeq(label.SessionId + "@" + ctx.Seq())
	ctx.VisitMeta(func(key, value []byte) {
		settings = append(settings, tp.WithAddMeta(string(key), string(value)))
	})
	if realIpBytes := ctx.PeekMeta(tp.MetaRealIp); len(realIpBytes) == 0 {
		label.RealIp = ctx.Ip()
		settings = append(settings, tp.WithAddMeta(tp.MetaRealIp, label.RealIp))
	} else {
		label.RealIp = goutil.BytesToString(realIpBytes)
	}
	label.Uri = ctx.Uri()
	rerr := p.pushCaller(&label).Push(label.Uri, ctx.InputBodyBytes(), settings...)
	if rerr != nil && rerr.Code < 200 && rerr.Code > 99 {
		rerr.Code = tp.CodeBadGateway
		rerr.Message = tp.CodeText(tp.CodeBadGateway)
	}
	return rerr
}
