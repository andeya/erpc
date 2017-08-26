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
		PostReg(*Handler) error
	}
	PostDialPlugin interface {
		PostDial(ForeSession) error
	}
	PostAcceptPlugin interface {
		PostAccept(ForeSession) error
	}
	PreWritePullPlugin interface {
		PreWritePull(WriteCtx) error
	}
	PostWritePullPlugin interface {
		PostWritePull(WriteCtx) error
	}
	PreWriteReplyPlugin interface {
		PreWriteReply(WriteCtx) error
	}
	PostWriteReplyPlugin interface {
		PostWriteReply(WriteCtx) error
	}
	PreWritePushPlugin interface {
		PreWritePush(WriteCtx) error
	}
	PostWritePushPlugin interface {
		PostWritePush(WriteCtx) error
	}
	PreReadHeaderPlugin interface {
		PreReadHeader(ReadCtx) error
	}
	PostReadHeaderPlugin interface {
		PostReadHeader(ReadCtx) error
	}
	PreReadBodyPlugin interface {
		PreReadBody(ReadCtx) error
	}
	PostReadBodyPlugin interface {
		PostReadBody(ReadCtx) error
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

func (p *pluginContainer) PostReg(h *Handler) error {
	var errs []error
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostRegPlugin); ok {
			err := _plugin.PostReg(h)
			if err != nil {
				errs = append(errs, errors.Errorf("PostRegPlugin(%s): %s", plugin.Name(), err.Error()))
			}
		}
	}
	return errors.Merge(errs...)
}

func (p *pluginContainer) PostDial(s ForeSession) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostDialPlugin); ok {
			if err := _plugin.PostDial(s); err != nil {
				s.Close()
				return errors.Errorf("PostDialPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostAccept(s ForeSession) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostAcceptPlugin); ok {
			if err := _plugin.PostAccept(s); err != nil {
				s.Close()
				return errors.Errorf("PostAcceptPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreWritePull(ctx WriteCtx) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreWritePullPlugin); ok {
			if err := _plugin.PreWritePull(ctx); err != nil {
				return errors.Errorf("PreWritePullPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostWritePull(ctx WriteCtx) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostWritePullPlugin); ok {
			if err := _plugin.PostWritePull(ctx); err != nil {
				return errors.Errorf("PostWritePullPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreWriteReply(ctx WriteCtx) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreWriteReplyPlugin); ok {
			if err := _plugin.PreWriteReply(ctx); err != nil {
				return errors.Errorf("PreWriteReplyPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostWriteReply(ctx WriteCtx) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostWriteReplyPlugin); ok {
			if err := _plugin.PostWriteReply(ctx); err != nil {
				return errors.Errorf("PostWriteReplyPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreWritePush(ctx WriteCtx) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreWritePushPlugin); ok {
			if err := _plugin.PreWritePush(ctx); err != nil {
				return errors.Errorf("PreWritePushPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostWritePush(ctx WriteCtx) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostWritePushPlugin); ok {
			if err := _plugin.PostWritePush(ctx); err != nil {
				return errors.Errorf("PostWritePushPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreReadHeader(ctx ReadCtx) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadHeaderPlugin); ok {
			if err := _plugin.PreReadHeader(ctx); err != nil {
				return errors.Errorf("PreReadHeaderPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostReadHeader(ctx ReadCtx) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadHeaderPlugin); ok {
			if err := _plugin.PostReadHeader(ctx); err != nil {
				return errors.Errorf("PostReadHeaderPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreReadBody(ctx ReadCtx) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadBodyPlugin); ok {
			if err := _plugin.PreReadBody(ctx); err != nil {
				return errors.Errorf("PreReadBodyPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostReadBody(ctx ReadCtx) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadBodyPlugin); ok {
			if err := _plugin.PostReadBody(ctx); err != nil {
				return errors.Errorf("PostReadBodyPlugin(%s): %s", plugin.Name(), err.Error())
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
