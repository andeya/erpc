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
		PostReg(*Handler) Xerror
	}
	PostDialPlugin interface {
		PostDial(ForeSession) Xerror
	}
	PostAcceptPlugin interface {
		PostAccept(ForeSession) Xerror
	}
	PreWritePullPlugin interface {
		PreWritePull(WriteCtx) Xerror
	}
	PostWritePullPlugin interface {
		PostWritePull(WriteCtx) Xerror
	}
	PreWriteReplyPlugin interface {
		PreWriteReply(WriteCtx) Xerror
	}
	PostWriteReplyPlugin interface {
		PostWriteReply(WriteCtx) Xerror
	}
	PreWritePushPlugin interface {
		PreWritePush(WriteCtx) Xerror
	}
	PostWritePushPlugin interface {
		PostWritePush(WriteCtx) Xerror
	}
	PreReadHeaderPlugin interface {
		PreReadHeader(ReadCtx) Xerror
	}
	PostReadHeaderPlugin interface {
		PostReadHeader(ReadCtx) Xerror
	}
	PreReadBodyPlugin interface {
		PreReadBody(ReadCtx) Xerror
	}
	PostReadBodyPlugin interface {
		PostReadBody(ReadCtx) Xerror
	}

	// PluginContainer plugin container that defines base methods to manage plugins.
	PluginContainer interface {
		Add(plugins ...Plugin) error
		Remove(pluginName string) error
		GetByName(pluginName string) Plugin
		GetAll() []Plugin

		PostDialPlugin
		PostAcceptPlugin
		PreWritePullPlugin
		PostWritePullPlugin
		PreWritePushPlugin
		PostWritePushPlugin
		PreReadHeaderPlugin
		PostReadHeaderPlugin

		PostRegPlugin

		PreWriteReplyPlugin
		PostWriteReplyPlugin
		PreReadBodyPlugin
		PostReadBodyPlugin

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

func (p *pluginContainer) PostReadHeader(ctx ReadCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadHeaderPlugin); ok {
			if xerr = _plugin.PostReadHeader(ctx); xerr != nil {
				Errorf("%s-PostReadHeaderPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreReadBody(ctx ReadCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadBodyPlugin); ok {
			if xerr = _plugin.PreReadBody(ctx); xerr != nil {
				Errorf("%s-PreReadBodyPlugin(%s)", plugin.Name(), xerr.Error())
				return xerr
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostReadBody(ctx ReadCtx) Xerror {
	var xerr Xerror
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadBodyPlugin); ok {
			if xerr = _plugin.PostReadBody(ctx); xerr != nil {
				Errorf("%s-PostReadBodyPlugin(%s)", plugin.Name(), xerr.Error())
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
		case PostReadHeaderPlugin:
			Warnf("invalid PostReadHeaderPlugin in router: %s", p.Name())
		case PreWritePushPlugin:
			Warnf("invalid PreWritePushPlugin in router: %s", p.Name())
		case PostWritePushPlugin:
			Warnf("invalid PostWritePushPlugin in router: %s", p.Name())
		}
	}
}
