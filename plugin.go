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

package tp

import (
	"net"

	"github.com/henrylee2cn/goutil/errors"
)

// Plug-ins during runtime

type (
	// Plugin plugin background
	Plugin interface {
		Name() string
	}
	// PreNewPeerPlugin is executed before creating peer.
	PreNewPeerPlugin interface {
		Plugin
		PreNewPeer(*PeerConfig, *PluginContainer) error
	}
	// PostNewPeerPlugin is executed after creating peer.
	PostNewPeerPlugin interface {
		Plugin
		PostNewPeer(EarlyPeer) error
	}
	// PostRegPlugin is executed after registering handler.
	PostRegPlugin interface {
		Plugin
		PostReg(*Handler) error
	}
	// PostListenPlugin is executed between listening and accepting.
	PostListenPlugin interface {
		Plugin
		PostListen() error
	}
	// PostDialPlugin is executed after dialing.
	PostDialPlugin interface {
		Plugin
		PostDial(EarlySession) *Rerror
	}
	// PostAcceptPlugin is executed after accepting connection.
	PostAcceptPlugin interface {
		Plugin
		PostAccept(EarlySession) *Rerror
	}
	// PreWritePullPlugin is executed before writing PULL packet.
	PreWritePullPlugin interface {
		Plugin
		PreWritePull(WriteCtx) *Rerror
	}
	// PostWritePullPlugin is executed after successful writing PULL packet.
	PostWritePullPlugin interface {
		Plugin
		PostWritePull(WriteCtx) *Rerror
	}
	// PreWriteReplyPlugin is executed before writing REPLY packet.
	PreWriteReplyPlugin interface {
		Plugin
		PreWriteReply(WriteCtx) *Rerror
	}
	// PostWriteReplyPlugin is executed after successful writing REPLY packet.
	PostWriteReplyPlugin interface {
		Plugin
		PostWriteReply(WriteCtx) *Rerror
	}
	// PreWritePushPlugin is executed before writing PUSH packet.
	PreWritePushPlugin interface {
		Plugin
		PreWritePush(WriteCtx) *Rerror
	}
	// PostWritePushPlugin is executed after successful writing PUSH packet.
	PostWritePushPlugin interface {
		Plugin
		PostWritePush(WriteCtx) *Rerror
	}
	// PreReadHeaderPlugin is executed before reading packet header.
	PreReadHeaderPlugin interface {
		Plugin
		PreReadHeader(PreCtx) *Rerror
	}
	// PostReadPullHeaderPlugin is executed after reading PULL packet header.
	PostReadPullHeaderPlugin interface {
		Plugin
		PostReadPullHeader(ReadCtx) *Rerror
	}
	// PreReadPullBodyPlugin is executed before reading PULL packet body.
	PreReadPullBodyPlugin interface {
		Plugin
		PreReadPullBody(ReadCtx) *Rerror
	}
	// PostReadPullBodyPlugin is executed after reading PULL packet body.
	PostReadPullBodyPlugin interface {
		Plugin
		PostReadPullBody(ReadCtx) *Rerror
	}
	// PostReadPushHeaderPlugin is executed after reading PUSH packet header.
	PostReadPushHeaderPlugin interface {
		Plugin
		PostReadPushHeader(ReadCtx) *Rerror
	}
	// PreReadPushBodyPlugin is executed before reading PUSH packet body.
	PreReadPushBodyPlugin interface {
		Plugin
		PreReadPushBody(ReadCtx) *Rerror
	}
	// PostReadPushBodyPlugin is executed after reading PUSH packet body.
	PostReadPushBodyPlugin interface {
		Plugin
		PostReadPushBody(ReadCtx) *Rerror
	}
	// PostReadReplyHeaderPlugin is executed after reading REPLY packet header.
	PostReadReplyHeaderPlugin interface {
		Plugin
		PostReadReplyHeader(ReadCtx) *Rerror
	}
	// PreReadReplyBodyPlugin is executed before reading REPLY packet body.
	PreReadReplyBodyPlugin interface {
		Plugin
		PreReadReplyBody(ReadCtx) *Rerror
	}
	// PostReadReplyBodyPlugin is executed after reading REPLY packet body.
	PostReadReplyBodyPlugin interface {
		Plugin
		PostReadReplyBody(ReadCtx) *Rerror
	}
	// PostDisconnectPlugin is executed after disconnection.
	PostDisconnectPlugin interface {
		Plugin
		PostDisconnect(BaseSession) *Rerror
	}
)

// newPluginContainer new a plugin container.
func newPluginContainer() *PluginContainer {
	return new(PluginContainer)
}

// PluginContainer plugins container.
type PluginContainer struct {
	plugins []Plugin
}

// AppendLeft appends plugins on the left side of the pluginContainer.
func (p *PluginContainer) AppendLeft(plugins ...Plugin) {
	if plugins == nil {
		plugins = make([]Plugin, 0)
	}
	if p.plugins == nil {
		p.plugins = make([]Plugin, 0)
	}
	for _, plugin := range p.plugins {
		if plugin == nil {
			Fatalf("plugin cannot be nil!")
			return
		}
		pName := plugin.Name()
		if len(pName) == 0 && p.GetByName(pName) != nil {
			Fatalf("repeat add plugin: %s", pName)
			return
		}
		plugins = append(plugins, plugin)
	}
	p.plugins = plugins
}

// AppendRight appends plugins on the right side of the pluginContainer.
func (p *PluginContainer) AppendRight(plugins ...Plugin) {
	if p.plugins == nil {
		p.plugins = make([]Plugin, 0)
	}
	for _, plugin := range plugins {
		if plugin == nil {
			Fatalf("plugin cannot be nil!")
			return
		}
		pName := plugin.Name()
		if len(pName) == 0 && p.GetByName(pName) != nil {
			Fatalf("repeat add plugin: %s", pName)
			return
		}
		p.plugins = append(p.plugins, plugin)
	}
}

func (p *PluginContainer) cloneAppendRight(plugins ...Plugin) *PluginContainer {
	clone := newPluginContainer()
	clone.AppendRight(p.GetAll()...)
	clone.AppendRight(plugins...)
	return clone
}

// Remove removes a plugin by it's name.
func (p *PluginContainer) Remove(pluginName string) error {
	if p.plugins == nil {
		return errors.New("no plugins are registered yet!")
	}
	if len(pluginName) == 0 {
		//return error: cannot delete an unamed plugin
		return errors.New("plugin with an empty name cannot be removed")
	}
	indexToRemove := -1
	for i, plugin := range p.plugins {
		if plugin.Name() == pluginName {
			indexToRemove = i
			break
		}
	}
	if indexToRemove == -1 {
		return errors.New("cannot remove a plugin which esn't exists")
	}
	p.plugins = append(p.plugins[:indexToRemove], p.plugins[indexToRemove+1:]...)
	return nil
}

// GetByName returns a plugin instance by it's name.
func (p *PluginContainer) GetByName(pluginName string) Plugin {
	if p.plugins == nil {
		return nil
	}
	for _, plugin := range p.plugins {
		if plugin.Name() == pluginName {
			return plugin
		}
	}
	return nil
}

// GetAll returns all activated plugins.
func (p *PluginContainer) GetAll() []Plugin {
	return p.plugins
}

// PreNewPeer executes the defined plugins before creating peer.
func (p *PluginContainer) PreNewPeer(peerConfig *PeerConfig) {
	var err error
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreNewPeerPlugin); ok {
			if err = _plugin.PreNewPeer(peerConfig, p); err != nil {
				Fatalf("%s-PreNewPeerPlugin(%s)", plugin.Name(), err.Error())
				return
			}
		}
	}
}

// PostNewPeer executes the defined plugins after creating peer.
func (p *PluginContainer) PostNewPeer(peer EarlyPeer) {
	var err error
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostNewPeerPlugin); ok {
			if err = _plugin.PostNewPeer(peer); err != nil {
				Fatalf("%s-PostNewPeerPlugin(%s)", plugin.Name(), err.Error())
				return
			}
		}
	}
}

// PostReg executes the defined plugins before registering handler.
func (p *PluginContainer) PostReg(h *Handler) {
	var err error
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostRegPlugin); ok {
			if err = _plugin.PostReg(h); err != nil {
				Fatalf("[register %s handler: %s] %s-PostRegPlugin(%s)", h.RouterTypeName(), h.Name(), plugin.Name(), err.Error())
				return
			}
		}
	}
}

// PostListen is executed between listening and accepting.
func (p *PluginContainer) PostListen(addr net.Addr) {
	var err error
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostListenPlugin); ok {
			if err = _plugin.PostListen(); err != nil {
				Fatalf("[network:%s, addr:%s] %s-PostListenPlugin(%s)", addr.Network(), addr.String(), plugin.Name(), err.Error())
				return
			}
		}
	}
	return
}

// PostDial executes the defined plugins after dialing.
func (p *PluginContainer) PostDial(sess EarlySession) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostDialPlugin); ok {
			if rerr = _plugin.PostDial(sess); rerr != nil {
				Debugf("[addr:%s, id:%s] %s-PostDialPlugin(%s)", sess.RemoteIp(), sess.Id(), plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

// PostAccept executes the defined plugins after accepting connection.
func (p *PluginContainer) PostAccept(sess EarlySession) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostAcceptPlugin); ok {
			if rerr = _plugin.PostAccept(sess); rerr != nil {
				Debugf("[addr:%s, id:%s] %s-PostAcceptPlugin(%s)", sess.RemoteIp(), sess.Id(), plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

// PreWritePull executes the defined plugins before writing PULL packet.
func (p *PluginContainer) PreWritePull(ctx WriteCtx) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreWritePullPlugin); ok {
			if rerr = _plugin.PreWritePull(ctx); rerr != nil {
				Debugf("%s-PreWritePullPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

// PostWritePull executes the defined plugins after successful writing PULL packet.
func (p *PluginContainer) PostWritePull(ctx WriteCtx) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostWritePullPlugin); ok {
			if rerr = _plugin.PostWritePull(ctx); rerr != nil {
				Errorf("%s-PostWritePullPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

// PreWriteReply executes the defined plugins before writing REPLY packet.
func (p *PluginContainer) PreWriteReply(ctx WriteCtx) {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreWriteReplyPlugin); ok {
			if rerr = _plugin.PreWriteReply(ctx); rerr != nil {
				Errorf("%s-PreWriteReplyPlugin(%s)", plugin.Name(), rerr.String())
				return
			}
		}
	}
}

// PostWriteReply executes the defined plugins after successful writing REPLY packet.
func (p *PluginContainer) PostWriteReply(ctx WriteCtx) {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostWriteReplyPlugin); ok {
			if rerr = _plugin.PostWriteReply(ctx); rerr != nil {
				Errorf("%s-PostWriteReplyPlugin(%s)", plugin.Name(), rerr.String())
				return
			}
		}
	}
}

// PreWritePush executes the defined plugins before writing PUSH packet.
func (p *PluginContainer) PreWritePush(ctx WriteCtx) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreWritePushPlugin); ok {
			if rerr = _plugin.PreWritePush(ctx); rerr != nil {
				Debugf("%s-PreWritePushPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

// PostWritePush executes the defined plugins after successful writing PUSH packet.
func (p *PluginContainer) PostWritePush(ctx WriteCtx) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostWritePushPlugin); ok {
			if rerr = _plugin.PostWritePush(ctx); rerr != nil {
				Errorf("%s-PostWritePushPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

// PreReadHeader executes the defined plugins before reading packet header.
func (p *PluginContainer) PreReadHeader(ctx PreCtx) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadHeaderPlugin); ok {
			if rerr = _plugin.PreReadHeader(ctx); rerr != nil {
				Debugf("disconnected when reading: %s-PreReadHeaderPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

// PostReadPullHeader executes the defined plugins after reading PULL packet header.
func (p *PluginContainer) PostReadPullHeader(ctx ReadCtx) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadPullHeaderPlugin); ok {
			if rerr = _plugin.PostReadPullHeader(ctx); rerr != nil {
				Errorf("%s-PostReadPullHeaderPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

// PreReadPullBody executes the defined plugins before reading PULL packet body.
func (p *PluginContainer) PreReadPullBody(ctx ReadCtx) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadPullBodyPlugin); ok {
			if rerr = _plugin.PreReadPullBody(ctx); rerr != nil {
				Errorf("%s-PreReadPullBodyPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

// PostReadPullBody executes the defined plugins after reading PULL packet body.
func (p *PluginContainer) PostReadPullBody(ctx ReadCtx) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadPullBodyPlugin); ok {
			if rerr = _plugin.PostReadPullBody(ctx); rerr != nil {
				Errorf("%s-PostReadPullBodyPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

// PostReadPushHeader executes the defined plugins after reading PUSH packet header.
func (p *PluginContainer) PostReadPushHeader(ctx ReadCtx) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadPushHeaderPlugin); ok {
			if rerr = _plugin.PostReadPushHeader(ctx); rerr != nil {
				Errorf("%s-PostReadPushHeaderPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

// PreReadPushBody executes the defined plugins before reading PUSH packet body.
func (p *PluginContainer) PreReadPushBody(ctx ReadCtx) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadPushBodyPlugin); ok {
			if rerr = _plugin.PreReadPushBody(ctx); rerr != nil {
				Errorf("%s-PreReadPushBodyPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

// PostReadPushBody executes the defined plugins after reading PUSH packet body.
func (p *PluginContainer) PostReadPushBody(ctx ReadCtx) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadPushBodyPlugin); ok {
			if rerr = _plugin.PostReadPushBody(ctx); rerr != nil {
				Errorf("%s-PostReadPushBodyPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

// PostReadReplyHeader executes the defined plugins after reading REPLY packet header.
func (p *PluginContainer) PostReadReplyHeader(ctx ReadCtx) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadReplyHeaderPlugin); ok {
			if rerr = _plugin.PostReadReplyHeader(ctx); rerr != nil {
				Errorf("%s-PostReadReplyHeaderPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

// PreReadReplyBody executes the defined plugins before reading REPLY packet body.
func (p *PluginContainer) PreReadReplyBody(ctx ReadCtx) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadReplyBodyPlugin); ok {
			if rerr = _plugin.PreReadReplyBody(ctx); rerr != nil {
				Errorf("%s-PreReadReplyBodyPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

// PostReadReplyBody executes the defined plugins after reading REPLY packet body.
func (p *PluginContainer) PostReadReplyBody(ctx ReadCtx) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadReplyBodyPlugin); ok {
			if rerr = _plugin.PostReadReplyBody(ctx); rerr != nil {
				Errorf("%s-PostReadReplyBodyPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

// PostDisconnect executes the defined plugins after disconnection.
func (p *PluginContainer) PostDisconnect(sess BaseSession) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostDisconnectPlugin); ok {
			if rerr = _plugin.PostDisconnect(sess); rerr != nil {
				Errorf("%s-PostDisconnectPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

func warnInvaildHandlerHooks(plugin []Plugin) {
	for _, p := range plugin {
		switch p.(type) {
		case PreNewPeerPlugin:
			Warnf("invalid PreNewPeerPlugin in router: %s", p.Name())
		case PostNewPeerPlugin:
			Warnf("invalid PostNewPeerPlugin in router: %s", p.Name())
		case PostDialPlugin:
			Warnf("invalid PostDialPlugin in router: %s", p.Name())
		case PostAcceptPlugin:
			Warnf("invalid PostAcceptPlugin in router: %s", p.Name())
		case PreWritePullPlugin:
			Warnf("invalid PreWritePullPlugin in router: %s", p.Name())
		case PostWritePullPlugin:
			Warnf("invalid PostWritePullPlugin in router: %s", p.Name())
		case PreWritePushPlugin:
			Warnf("invalid PreWritePushPlugin in router: %s", p.Name())
		case PostWritePushPlugin:
			Warnf("invalid PostWritePushPlugin in router: %s", p.Name())
		case PreReadHeaderPlugin:
			Warnf("invalid PreReadHeaderPlugin in router: %s", p.Name())
		case PostReadPullHeaderPlugin:
			Warnf("invalid PostReadPullHeaderPlugin in router: %s", p.Name())
		case PostReadPushHeaderPlugin:
			Warnf("invalid PostReadPushHeaderPlugin in router: %s", p.Name())
		}
	}
}
