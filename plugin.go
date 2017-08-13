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

package teleport

import (
	"github.com/henrylee2cn/goutil/errors"

	"github.com/henrylee2cn/teleport/socket"
)

// Interfaces about plugin.
type (
	Plugin interface {
		Name() string
	}
	PreConnectPlugin interface {
		PreConnect(string, interface{}) error
	}
	PostConnectPlugin interface {
		PostConnect(socket.Socket) error
	}
	PreWritePacketPlugin interface {
		PreWritePacket(Context, interface{}) error
	}
	PostWritePacketPlugin interface {
		PostWritePacket(Context, interface{}) error
	}
	PreReadHeaderPlugin interface {
		PreReadHeader(Context) error
	}
	PostReadHeaderPlugin interface {
		PostReadHeader(Context) error
	}
	PreReadBodyPlugin interface {
		PreReadBody(Context, interface{}) error
	}
	PostReadBodyPlugin interface {
		PostReadBody(Context, interface{}) error
	}
	// PluginContainer plugin container that defines base methods to manage plugins.
	PluginContainer interface {
		Add(plugins ...Plugin) error
		Remove(pluginName string) error
		GetByName(pluginName string) Plugin
		GetAll() []Plugin
		PreConnectPlugin
		PostConnectPlugin
		PreWritePacketPlugin
		PostWritePacketPlugin
		PreReadHeaderPlugin
		PostReadHeaderPlugin
		PreReadBodyPlugin
		PostReadBodyPlugin
	}
)

// NewPluginContainer new a plugin container.
func NewPluginContainer() PluginContainer {
	return new(pluginContainer)
}

type pluginContainer struct {
	plugins []Plugin
}

// Add adds a plugin.
func (p *pluginContainer) Add(plugins ...Plugin) error {
	if p.plugins == nil {
		p.plugins = make([]Plugin, 0)
	}
	for _, plugin := range plugins {
		if plugin == nil {
			return errors.New("the plugin cannot be nil!")
		}
		pName := plugin.Name()
		if len(pName) == 0 && p.GetByName(pName) != nil {
			return errors.Errorf("repeat add the plugin: %s", pName)
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

func (p *pluginContainer) PreConnect(nodePath string, rcvr interface{}) error {
	var errs []error
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreConnectPlugin); ok {
			err := _plugin.PreConnect(nodePath, rcvr)
			if err != nil {
				errs = append(errs, errors.Errorf("PreConnectPlugin(%s): %s", plugin.Name(), err.Error()))
			}
		}
	}
	return errors.Merge(errs...)
}

func (p *pluginContainer) PostConnect(s socket.Socket) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostConnectPlugin); ok {
			if err := _plugin.PostConnect(s); err != nil {
				s.Close()
				return errors.Errorf("PostConnectPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreWritePacket(ctx Context, body interface{}) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreWritePacketPlugin); ok {
			if err := _plugin.PreWritePacket(ctx, body); err != nil {
				return errors.Errorf("PreWritePacketPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostWritePacket(ctx Context, body interface{}) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostWritePacketPlugin); ok {
			if err := _plugin.PostWritePacket(ctx, body); err != nil {
				return errors.Errorf("PostWritePacketPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreReadHeader(ctx Context) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadHeaderPlugin); ok {
			if err := _plugin.PreReadHeader(ctx); err != nil {
				return errors.Errorf("PreReadHeaderPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostReadHeader(ctx Context) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadHeaderPlugin); ok {
			if err := _plugin.PostReadHeader(ctx); err != nil {
				return errors.Errorf("PostReadHeaderPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PreReadBody(ctx Context, body interface{}) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadBodyPlugin); ok {
			if err := _plugin.PreReadBody(ctx, body); err != nil {
				return errors.Errorf("PreReadBodyPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) PostReadBody(ctx Context, body interface{}) error {
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadBodyPlugin); ok {
			if err := _plugin.PostReadBody(ctx, body); err != nil {
				return errors.Errorf("PostReadBodyPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}
