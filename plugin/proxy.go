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
func Proxy(caller Caller) tp.Plugin {
	return &proxy{
		pullFunc: caller.Pull,
		pushFunc: caller.Push,
	}
}

// ProxyPull creates a proxy plugin for handling unknown pulling.
func ProxyPull(fn PullFunc) tp.Plugin {
	return &proxy{pullFunc: fn}
}

// ProxyPush creates a proxy plugin for handling unknown pushing.
func ProxyPush(fn PushFunc) tp.Plugin {
	return &proxy{pushFunc: fn}
}

type (
	// Caller the object used to pull and push
	Caller interface {
		Pull(uri string, args interface{}, reply interface{}, setting ...socket.PacketSetting) tp.PullCmd
		Push(uri string, args interface{}, setting ...socket.PacketSetting) *tp.Rerror
	}
	// PullFunc the function used to pull
	PullFunc func(uri string, args interface{}, reply interface{}, setting ...socket.PacketSetting) tp.PullCmd
	// PushFunc the function used to push
	PushFunc func(uri string, args interface{}, setting ...socket.PacketSetting) *tp.Rerror
	proxy    struct {
		peer     tp.Peer
		pullFunc PullFunc
		pushFunc PushFunc
	}
)

var (
	_ tp.PostNewPeerPlugin = new(proxy)
	_ tp.PostListenPlugin  = new(proxy)
)

func (p *proxy) Name() string {
	return "proxy"
}

func (p *proxy) PostNewPeer(peer tp.EarlyPeer) error {
	p.peer = peer.(tp.Peer)
	return nil
}

func (p *proxy) PostListen() error {
	if p.pullFunc != nil {
		p.peer.SetUnknownPull(p.pull)
	}
	if p.pushFunc != nil {
		p.peer.SetUnknownPush(p.push)
	}
	return nil
}

func (p *proxy) pull(ctx tp.UnknownPullCtx) (interface{}, *tp.Rerror) {
	var settings = make([]socket.PacketSetting, 0, 8)
	ctx.VisitMeta(func(key, value []byte) {
		settings = append(settings, socket.WithAddMeta(string(key), string(value)))
	})
	if len(ctx.PeekMeta(tp.MetaRealIp)) == 0 {
		settings = append(settings, socket.WithAddMeta(tp.MetaRealIp, ctx.Ip()))
	}
	var reply []byte
	pullcmd := p.pullFunc(ctx.Uri(), ctx.InputBodyBytes(), &reply, settings...)
	pullcmd.InputMeta().VisitAll(func(key, value []byte) {
		ctx.SetMeta(goutil.BytesToString(key), goutil.BytesToString(value))
	})
	return reply, pullcmd.Rerror()
}

func (p *proxy) push(ctx tp.UnknownPushCtx) *tp.Rerror {
	var settings = make([]socket.PacketSetting, 0, 8)
	ctx.VisitMeta(func(key, value []byte) {
		settings = append(settings, socket.WithAddMeta(string(key), string(value)))
	})
	if len(ctx.PeekMeta(tp.MetaRealIp)) == 0 {
		settings = append(settings, socket.WithAddMeta(tp.MetaRealIp, ctx.Ip()))
	}
	return p.pushFunc(ctx.Uri(), ctx.InputBodyBytes(), settings...)
}
