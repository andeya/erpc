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
	"github.com/henrylee2cn/goutil/errors"
)

// Interfaces about plugin.
type (
	Plugin interface {
		Name() string
	}
	PostRegPlugin interface {
		Plugin
		PostReg(*Handler) *Rerror
	}
	PostDialPlugin interface {
		Plugin
		PostDial(PreSession) *Rerror
	}
	PostAcceptPlugin interface {
		Plugin
		PostAccept(PreSession) *Rerror
	}
	PreWritePullPlugin interface {
		Plugin
		PreWritePull(WriteCtx) *Rerror
	}
	PostWritePullPlugin interface {
		Plugin
		PostWritePull(WriteCtx) *Rerror
	}
	PreWriteReplyPlugin interface {
		Plugin
		PreWriteReply(WriteCtx) *Rerror
	}
	PostWriteReplyPlugin interface {
		Plugin
		PostWriteReply(WriteCtx) *Rerror
	}
	PreWritePushPlugin interface {
		Plugin
		PreWritePush(WriteCtx) *Rerror
	}
	PostWritePushPlugin interface {
		Plugin
		PostWritePush(WriteCtx) *Rerror
	}
	PreReadHeaderPlugin interface {
		Plugin
		PreReadHeader(ReadCtx) *Rerror
	}

	PostReadPullHeaderPlugin interface {
		Plugin
		PostReadPullHeader(ReadCtx) *Rerror
	}
	PreReadPullBodyPlugin interface {
		Plugin
		PreReadPullBody(ReadCtx) *Rerror
	}
	PostReadPullBodyPlugin interface {
		Plugin
		PostReadPullBody(ReadCtx) *Rerror
	}

	PostReadPushHeaderPlugin interface {
		Plugin
		PostReadPushHeader(ReadCtx) *Rerror
	}
	PreReadPushBodyPlugin interface {
		Plugin
		PreReadPushBody(ReadCtx) *Rerror
	}
	PostReadPushBodyPlugin interface {
		Plugin
		PostReadPushBody(ReadCtx) *Rerror
	}

	PostReadReplyHeaderPlugin interface {
		Plugin
		PostReadReplyHeader(ReadCtx) *Rerror
	}
	PreReadReplyBodyPlugin interface {
		Plugin
		PreReadReplyBody(ReadCtx) *Rerror
	}
	PostReadReplyBodyPlugin interface {
		Plugin
		PostReadReplyBody(ReadCtx) *Rerror
	}

	PostDisconnectPlugin interface {
		Plugin
		PostDisconnect(PostSession) *Rerror
	}

	// PluginContainer plugin container that defines base methods to manage plugins.
	PluginContainer interface {
		Add(plugins ...Plugin) error
		Remove(pluginName string) error
		GetByName(pluginName string) Plugin
		GetAll() []Plugin

		PostReg(*Handler) *Rerror
		PostDial(PreSession) *Rerror
		PostAccept(PreSession) *Rerror
		PreWritePull(WriteCtx) *Rerror
		PostWritePull(WriteCtx) *Rerror
		PreWriteReply(WriteCtx) *Rerror
		PostWriteReply(WriteCtx) *Rerror
		PreWritePush(WriteCtx) *Rerror
		PostWritePush(WriteCtx) *Rerror
		PreReadHeader(ReadCtx) *Rerror

		PostReadPullHeader(ReadCtx) *Rerror
		PreReadPullBody(ReadCtx) *Rerror
		PostReadPullBody(ReadCtx) *Rerror

		PostReadPushHeader(ReadCtx) *Rerror
		PreReadPushBody(ReadCtx) *Rerror
		PostReadPushBody(ReadCtx) *Rerror

		PostReadReplyHeader(ReadCtx) *Rerror
		PreReadReplyBody(ReadCtx) *Rerror
		PostReadReplyBody(ReadCtx) *Rerror

		PostDisconnect(PostSession) *Rerror

		cloneAdd(...Plugin) (PluginContainer, error)
	}
)

// newPluginContainer new a plugin container.
func newPluginContainer() PluginContainer {
	return new(pluginContainer)
}

type pluginContainer struct {
	plugins []Plugin
}

func (p *pluginContainer) cloneAdd(plugins ...Plugin) (PluginContainer, error) {
	clone := newPluginContainer()
	err := clone.Add(p.GetAll()...)
	if err != nil {
		return nil, err
	}
	err = clone.Add(plugins...)
	if err != nil {
		return nil, err
	}
	return clone, nil
}

// Add adds a plugin.
func (p *pluginContainer) Add(plugins ...Plugin) error {
	if p.plugins == nil {
		p.plugins = make([]Plugin, 0)
	}
	for _, plugin := range plugins {
		if plugin == nil {
			return errors.New("plugin cannot be nil!")
		}
		pName := plugin.Name()
		if len(pName) == 0 && p.GetByName(pName) != nil {
			return errors.Errorf("repeat add plugin: %s", pName)
		}
		p.plugins = append(p.plugins, plugin)
	}
	return nil
}

// Remove removes a plugin by it's name.
func (p *pluginContainer) Remove(pluginName string) error {
	if p.plugins == nil {
		return errors.New("no plugins are registed yet!")
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

// GetByName returns a plugin instance by it's name
func (p *pluginContainer) GetByName(pluginName string) Plugin {
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

// GetAll returns all activated plugins
func (p *pluginContainer) GetAll() []Plugin {
	return p.plugins
}

func (p *pluginContainer) PostReg(h *Handler) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostRegPlugin); ok {
			if rerr = _plugin.PostReg(h); rerr != nil {
				Fatalf("%s-PostRegPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostDial(sess PreSession) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostDialPlugin); ok {
			if rerr = _plugin.PostDial(sess); rerr != nil {
				Debugf("dial fail (addr: %s, id: %s): %s-PostDialPlugin(%s)", sess.RemoteIp(), sess.Id(), plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostAccept(sess PreSession) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostAcceptPlugin); ok {
			if rerr = _plugin.PostAccept(sess); rerr != nil {
				Debugf("accept session(addr: %s, id: %s): %s-PostAcceptPlugin(%s)", sess.RemoteIp(), sess.Id(), plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreWritePull(ctx WriteCtx) *Rerror {
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

func (p *pluginContainer) PostWritePull(ctx WriteCtx) *Rerror {
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

func (p *pluginContainer) PreWriteReply(ctx WriteCtx) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreWriteReplyPlugin); ok {
			if rerr = _plugin.PreWriteReply(ctx); rerr != nil {
				Errorf("%s-PreWriteReplyPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostWriteReply(ctx WriteCtx) *Rerror {
	var rerr *Rerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostWriteReplyPlugin); ok {
			if rerr = _plugin.PostWriteReply(ctx); rerr != nil {
				Errorf("%s-PostWriteReplyPlugin(%s)", plugin.Name(), rerr.String())
				return rerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreWritePush(ctx WriteCtx) *Rerror {
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

func (p *pluginContainer) PostWritePush(ctx WriteCtx) *Rerror {
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

func (p *pluginContainer) PreReadHeader(ctx ReadCtx) *Rerror {
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

func (p *pluginContainer) PostReadPullHeader(ctx ReadCtx) *Rerror {
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

func (p *pluginContainer) PreReadPullBody(ctx ReadCtx) *Rerror {
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

func (p *pluginContainer) PostReadPullBody(ctx ReadCtx) *Rerror {
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

func (p *pluginContainer) PostReadPushHeader(ctx ReadCtx) *Rerror {
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

func (p *pluginContainer) PreReadPushBody(ctx ReadCtx) *Rerror {
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

func (p *pluginContainer) PostReadPushBody(ctx ReadCtx) *Rerror {
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

func (p *pluginContainer) PostReadReplyHeader(ctx ReadCtx) *Rerror {
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

func (p *pluginContainer) PreReadReplyBody(ctx ReadCtx) *Rerror {
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

func (p *pluginContainer) PostReadReplyBody(ctx ReadCtx) *Rerror {
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

func (p *pluginContainer) PostDisconnect(sess PostSession) *Rerror {
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

func warnInvaildRouterHooks(plugin []Plugin) {
	for _, p := range plugin {
		switch p.(type) {
		case PostDialPlugin:
			Warnf("invalid PostDialPlugin in router: %s", p.Name())
		case PostAcceptPlugin:
			Warnf("invalid PostAcceptPlugin in router: %s", p.Name())
		case PreWritePullPlugin:
			Warnf("invalid PreWritePullPlugin in router: %s", p.Name())
		case PostWritePullPlugin:
			Warnf("invalid PostWritePullPlugin in router: %s", p.Name())
		case PreReadHeaderPlugin:
			Warnf("invalid PreReadHeaderPlugin in router: %s", p.Name())
		case PostReadPullHeaderPlugin:
			Warnf("invalid PostReadPullHeaderPlugin in router: %s", p.Name())
		case PreWritePushPlugin:
			Warnf("invalid PreWritePushPlugin in router: %s", p.Name())
		case PostWritePushPlugin:
			Warnf("invalid PostWritePushPlugin in router: %s", p.Name())
		}
	}
}
