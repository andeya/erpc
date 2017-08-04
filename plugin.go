package teleport

import (
	"github.com/henrylee2cn/goutil/errors"
)

type (
	// PluginContainer plugin container that defines base methods to manage plugins.
	PluginContainer interface {
		Add(plugins ...Plugin) error
		Remove(pluginName string) error
		GetByName(pluginName string) Plugin
		GetAll() []Plugin
		doPostConnect(Conn) error
		doPreReadHeader(Context) error
		doPostReadHeader(Context) error
		doPreReadBody(Context, interface{}) error
		doPostReadBody(Context, interface{}) error
		doPreWritePacket(Context, interface{}) error
		doPostWritePacket(Context, interface{}) error
	}
	//Plugin represents a plugin.
	Plugin interface {
		Name() string
	}
	PreConnectPlugin interface {
		Plugin
		PreConnect(nodePath string, rcvr interface{}, metadata ...string) error
	}
	PostConnectPlugin interface {
		Plugin
		PostConnect(Conn) error
	}
	PreWritePacketPlugin interface {
		Plugin
		PreWritePacket(Context, interface{}) error
	}
	PostWritePacketPlugin interface {
		Plugin
		PostWritePacket(Context, interface{}) error
	}
	PreReadHeaderPlugin interface {
		Plugin
		PreReadHeader(Context) error
	}
	PostReadHeaderPlugin interface {
		Plugin
		PostReadHeader(Context) error
	}
	PreReadBodyPlugin interface {
		Plugin
		PreReadBody(interface{}) error
	}
	PostReadBodyPlugin interface {
		Plugin
		PostReadBody(interface{}) error
	}
)

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
		return errors.New("cannot remove a plugin which doesn't exists")
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

func (p *pluginContainer) doPreConnect(nodePath string, rcvr interface{}, metadata ...string) error {
	var errs []error
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PreConnectPlugin); ok {
			err := plugin.PreConnect(nodePath, rcvr, metadata...)
			if err != nil {
				errs = append(errs, errors.Errorf("PreConnectPlugin(%s): %s", plugin.Name(), err.Error()))
			}
		}
	}
	return errors.Merge(errs...)
}

func (p *pluginContainer) doPostConnect(conn Conn) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PostConnectPlugin); ok {
			if err := plugin.PostConnect(conn); err != nil {
				conn.Close()
				return errors.Errorf("PostConnectPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPreWritePacket(r Context, body interface{}) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PreWritePacketPlugin); ok {
			if err := plugin.PreWritePacket(r, body); err != nil {
				return errors.Errorf("PreWritePacketPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPostWritePacket(r Context, body interface{}) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PostWritePacketPlugin); ok {
			if err := plugin.PostWritePacket(r, body); err != nil {
				return errors.Errorf("PostWritePacketPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPreReadHeader(r Context) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PreReadHeaderPlugin); ok {
			if err := plugin.PreReadHeader(r); err != nil {
				return errors.Errorf("PreReadHeaderPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPostReadHeader(r Context) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PostReadHeaderPlugin); ok {
			if err := plugin.PostReadHeader(r); err != nil {
				return errors.Errorf("PostReadHeaderPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPreReadBody(body interface{}) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PreReadBodyPlugin); ok {
			if err := plugin.PreReadBody(body); err != nil {
				return errors.Errorf("PreReadBodyPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPostReadBody(body interface{}) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PostReadBodyPlugin); ok {
			if err := plugin.PostReadBody(body); err != nil {
				return errors.Errorf("PostReadBodyPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}
