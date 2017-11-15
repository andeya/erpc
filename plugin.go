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
		PostReg(*Handler) Xerror
	}
	PostDialPlugin interface {
		Plugin
		PostDial(ForeSession) Xerror
	}
	PostAcceptPlugin interface {
		Plugin
		PostAccept(ForeSession) Xerror
	}
	PreWritePullPlugin interface {
		Plugin
		PreWritePull(WriteCtx) Xerror
	}
	PostWritePullPlugin interface {
		Plugin
		PostWritePull(WriteCtx) Xerror
	}
	PreWriteReplyPlugin interface {
		Plugin
		PreWriteReply(WriteCtx) Xerror
	}
	PostWriteReplyPlugin interface {
		Plugin
		PostWriteReply(WriteCtx) Xerror
	}
	PreWritePushPlugin interface {
		Plugin
		PreWritePush(WriteCtx) Xerror
	}
	PostWritePushPlugin interface {
		Plugin
		PostWritePush(WriteCtx) Xerror
	}
	PreReadHeaderPlugin interface {
		Plugin
		PreReadHeader(ReadCtx) Xerror
	}

	PostReadPullHeaderPlugin interface {
		Plugin
		PostReadPullHeader(ReadCtx) Xerror
	}
	PreReadPullBodyPlugin interface {
		Plugin
		PreReadPullBody(ReadCtx) Xerror
	}
	PostReadPullBodyPlugin interface {
		Plugin
		PostReadPullBody(ReadCtx) Xerror
	}

	PostReadPushHeaderPlugin interface {
		Plugin
		PostReadPushHeader(ReadCtx) Xerror
	}
	PreReadPushBodyPlugin interface {
		Plugin
		PreReadPushBody(ReadCtx) Xerror
	}
	PostReadPushBodyPlugin interface {
		Plugin
		PostReadPushBody(ReadCtx) Xerror
	}

	PostReadReplyHeaderPlugin interface {
		Plugin
		PostReadReplyHeader(ReadCtx) Xerror
	}
	PreReadReplyBodyPlugin interface {
		Plugin
		PreReadReplyBody(ReadCtx) Xerror
	}
	PostReadReplyBodyPlugin interface {
		Plugin
		PostReadReplyBody(ReadCtx) Xerror
	}

	// PluginContainer plugin container that defines base methods to manage plugins.
	PluginContainer interface {
		Add(plugins ...Plugin) error
		Remove(pluginName string) error
		GetByName(pluginName string) Plugin
		GetAll() []Plugin

		PostReg(*Handler) Xerror
		PostDial(ForeSession) Xerror
		PostAccept(ForeSession) Xerror
		PreWritePull(WriteCtx) Xerror
		PostWritePull(WriteCtx) Xerror
		PreWriteReply(WriteCtx) Xerror
		PostWriteReply(WriteCtx) Xerror
		PreWritePush(WriteCtx) Xerror
		PostWritePush(WriteCtx) Xerror
		PreReadHeader(ReadCtx) Xerror

		PostReadPullHeader(ReadCtx) Xerror
		PreReadPullBody(ReadCtx) Xerror
		PostReadPullBody(ReadCtx) Xerror

		PostReadPushHeader(ReadCtx) Xerror
		PreReadPushBody(ReadCtx) Xerror
		PostReadPushBody(ReadCtx) Xerror

		PostReadReplyHeader(ReadCtx) Xerror
		PreReadReplyBody(ReadCtx) Xerror
		PostReadReplyBody(ReadCtx) Xerror

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

func (p *pluginContainer) PostReg(h *Handler) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostRegPlugin); ok {
			if xerr = _plugin.PostReg(h); xerr != nil {
				Fatalf("%s-PostRegPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostDial(sess ForeSession) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostDialPlugin); ok {
			if xerr = _plugin.PostDial(sess); xerr != nil {
				Debugf("dial fail (addr: %s, id: %s): %s-PostDialPlugin(%s)", sess.RemoteIp(), sess.Id(), plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostAccept(sess ForeSession) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostAcceptPlugin); ok {
			if xerr = _plugin.PostAccept(sess); xerr != nil {
				Debugf("accept session(addr: %s, id: %s): %s-PostAcceptPlugin(%s)", sess.RemoteIp(), sess.Id(), plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreWritePull(ctx WriteCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreWritePullPlugin); ok {
			if xerr = _plugin.PreWritePull(ctx); xerr != nil {
				Debugf("%s-PreWritePullPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostWritePull(ctx WriteCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostWritePullPlugin); ok {
			if xerr = _plugin.PostWritePull(ctx); xerr != nil {
				Errorf("%s-PostWritePullPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreWriteReply(ctx WriteCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreWriteReplyPlugin); ok {
			if xerr = _plugin.PreWriteReply(ctx); xerr != nil {
				Errorf("%s-PreWriteReplyPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostWriteReply(ctx WriteCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostWriteReplyPlugin); ok {
			if xerr = _plugin.PostWriteReply(ctx); xerr != nil {
				Errorf("%s-PostWriteReplyPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreWritePush(ctx WriteCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreWritePushPlugin); ok {
			if xerr = _plugin.PreWritePush(ctx); xerr != nil {
				Debugf("%s-PreWritePushPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostWritePush(ctx WriteCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostWritePushPlugin); ok {
			if xerr = _plugin.PostWritePush(ctx); xerr != nil {
				Debugf("%s-PostWritePushPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreReadHeader(ctx ReadCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadHeaderPlugin); ok {
			if xerr = _plugin.PreReadHeader(ctx); xerr != nil {
				Debugf("disconnected when reading: %s-PreReadHeaderPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostReadPullHeader(ctx ReadCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadPullHeaderPlugin); ok {
			if xerr = _plugin.PostReadPullHeader(ctx); xerr != nil {
				Errorf("%s-PostReadPullHeaderPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreReadPullBody(ctx ReadCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadPullBodyPlugin); ok {
			if xerr = _plugin.PreReadPullBody(ctx); xerr != nil {
				Errorf("%s-PreReadPullBodyPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostReadPullBody(ctx ReadCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadPullBodyPlugin); ok {
			if xerr = _plugin.PostReadPullBody(ctx); xerr != nil {
				Errorf("%s-PostReadPullBodyPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostReadPushHeader(ctx ReadCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadPushHeaderPlugin); ok {
			if xerr = _plugin.PostReadPushHeader(ctx); xerr != nil {
				Errorf("%s-PostReadPushHeaderPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreReadPushBody(ctx ReadCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadPushBodyPlugin); ok {
			if xerr = _plugin.PreReadPushBody(ctx); xerr != nil {
				Errorf("%s-PreReadPushBodyPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostReadPushBody(ctx ReadCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadPushBodyPlugin); ok {
			if xerr = _plugin.PostReadPushBody(ctx); xerr != nil {
				Errorf("%s-PostReadPushBodyPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostReadReplyHeader(ctx ReadCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadReplyHeaderPlugin); ok {
			if xerr = _plugin.PostReadReplyHeader(ctx); xerr != nil {
				Errorf("%s-PostReadReplyHeaderPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreReadReplyBody(ctx ReadCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadReplyBodyPlugin); ok {
			if xerr = _plugin.PreReadReplyBody(ctx); xerr != nil {
				Errorf("%s-PreReadReplyBodyPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostReadReplyBody(ctx ReadCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadReplyBodyPlugin); ok {
			if xerr = _plugin.PostReadReplyBody(ctx); xerr != nil {
				Errorf("%s-PostReadReplyBodyPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
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
