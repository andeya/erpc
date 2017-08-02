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
	}
	//Plugin represents a plugin.
	Plugin interface {
		Name() string
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
	for _, _plugin := range p.plugins {
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

// client plugin interfaces.
type (
	// CliPluginContainer defines all methods to manage client plugins.
	CliPluginContainer interface {
		PluginContainer
		doPostConnected(CliCodecConn) error
		doPreWriteRequest(*common.Request, interface{}) error
		doPostWriteRequest(*common.Request, interface{}) error
		doPreReadResponseHeader(*common.Response) error
		doPostReadResponseHeader(*common.Response) error
		doPreReadResponseBody(interface{}) error
		doPostReadResponseBody(interface{}) error
	}
	PostConnectedPlugin interface {
		Plugin
		PostConnected(CliCodecConn) error
	}
	PreWriteRequestPlugin interface {
		Plugin
		PreWriteRequest(*common.Request, interface{}) error
	}
	PostWriteRequestPlugin interface {
		Plugin
		PostWriteRequest(*common.Request, interface{}) error
	}
	PreReadResponseHeaderPlugin interface {
		Plugin
		PreReadResponseHeader(*common.Response) error
	}
	PostReadResponseHeaderPlugin interface {
		Plugin
		PostReadResponseHeader(*common.Response) error
	}
	PreReadResponseBodyPlugin interface {
		Plugin
		PreReadResponseBody(interface{}) error
	}
	PostReadResponseBodyPlugin interface {
		Plugin
		PostReadResponseBody(interface{}) error
	}
)

// NewCliPluginContainer creates a new client plugin container.
func NewCliPluginContainer() CliPluginContainer {
	return new(pluginContainer)
}

func (p *pluginContainer) doPostConnected(codecConn CliCodecConn) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PostConnectedPlugin); ok {
			if err := plugin.PostConnected(codecConn); err != nil {
				codecConn.Close()
				return errors.Errorf("PostConnectedPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPreWriteRequest(r *common.Request, body interface{}) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PreWriteRequestPlugin); ok {
			if err := plugin.PreWriteRequest(r, body); err != nil {
				return errors.Errorf("PreWriteRequestPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPostWriteRequest(r *common.Request, body interface{}) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PostWriteRequestPlugin); ok {
			if err := plugin.PostWriteRequest(r, body); err != nil {
				return errors.Errorf("PostWriteRequestPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPreReadResponseHeader(r *common.Response) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PreReadResponseHeaderPlugin); ok {
			if err := plugin.PreReadResponseHeader(r); err != nil {
				return errors.Errorf("PreReadResponseHeaderPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPostReadResponseHeader(r *common.Response) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PostReadResponseHeaderPlugin); ok {
			if err := plugin.PostReadResponseHeader(r); err != nil {
				return errors.Errorf("PostReadResponseHeaderPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPreReadResponseBody(body interface{}) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PreReadResponseBodyPlugin); ok {
			if err := plugin.PreReadResponseBody(body); err != nil {
				return errors.Errorf("PreReadResponseBodyPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPostReadResponseBody(body interface{}) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PostReadResponseBodyPlugin); ok {
			if err := plugin.PostReadResponseBody(body); err != nil {
				return errors.Errorf("PostReadResponseBodyPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

// server plugin interfaces.
type (
	// SvrPluginContainer defines all methods to manage server plugins.
	SvrPluginContainer interface {
		PluginContainer
		doRegister(nodePath string, rcvr interface{}, metadata ...string) error
		doPostConnAccept(SvrCodecConn) error
		doPreReadRequestHeader(*Context) error
		doPostReadRequestHeader(*Context) error
		doPreReadRequestBody(ctx *Context, body interface{}) error
		doPostReadRequestBody(ctx *Context, body interface{}) error
		doPreWriteResponse(ctx *Context, body interface{}) error
		doPostWriteResponse(ctx *Context, body interface{}) error
	}
	RegisterPlugin interface {
		Plugin
		Register(nodePath string, rcvr interface{}, metadata ...string) error
	}
	//PostConnAcceptPlugin is connection accept plugin.
	// if returns error, it means subsequent PostConnAcceptPlugins should not contiune to handle this conn
	// and this conn has been closed.
	PostConnAcceptPlugin interface {
		Plugin
		PostConnAccept(SvrCodecConn) error
	}
	PreReadRequestHeaderPlugin interface {
		Plugin
		PreReadRequestHeader(*Context) error
	}
	PostReadRequestHeaderPlugin interface {
		Plugin
		PostReadRequestHeader(*Context) error
	}
	PreReadRequestBodyPlugin interface {
		Plugin
		PreReadRequestBody(ctx *Context, body interface{}) error
	}
	PostReadRequestBodyPlugin interface {
		Plugin
		PostReadRequestBody(ctx *Context, body interface{}) error
	}
	PreWriteResponsePlugin interface {
		Plugin
		PreWriteResponse(ctx *Context, body interface{}) error
	}
	PostWriteResponsePlugin interface {
		Plugin
		PostWriteResponse(ctx *Context, body interface{}) error
	}
)

// NewSvrPluginContainer creates a new server plugin container.
func NewSvrPluginContainer() SvrPluginContainer {
	return new(pluginContainer)
}

func (p *pluginContainer) doRegister(nodePath string, rcvr interface{}, metadata ...string) error {
	var errs []error
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(RegisterPlugin); ok {
			err := plugin.Register(nodePath, rcvr, metadata...)
			if err != nil {
				errs = append(errs, errors.Errorf("RegisterPlugin(%s): %s", plugin.Name(), err.Error()))
			}
		}
	}
	return errors.Merge(errs...)
}

func (p *pluginContainer) doPostConnAccept(conn SvrCodecConn) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PostConnAcceptPlugin); ok {
			if err := plugin.PostConnAccept(conn); err != nil {
				conn.Close()
				return errors.Errorf("PostConnAcceptPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPreReadRequestHeader(ctx *Context) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PreReadRequestHeaderPlugin); ok {
			if err := plugin.PreReadRequestHeader(ctx); err != nil {
				return errors.Errorf("PreReadRequestHeaderPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPostReadRequestHeader(ctx *Context) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PostReadRequestHeaderPlugin); ok {
			if err := plugin.PostReadRequestHeader(ctx); err != nil {
				return errors.Errorf("PostReadRequestHeaderPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPreReadRequestBody(ctx *Context, body interface{}) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PreReadRequestBodyPlugin); ok {
			if err := plugin.PreReadRequestBody(ctx, body); err != nil {
				return errors.Errorf("PreReadRequestBodyPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPostReadRequestBody(ctx *Context, body interface{}) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PostReadRequestBodyPlugin); ok {
			if err := plugin.PostReadRequestBody(ctx, body); err != nil {
				return errors.Errorf("PostReadRequestBodyPlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPreWriteResponse(ctx *Context, body interface{}) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PreWriteResponsePlugin); ok {
			if err := plugin.PreWriteResponse(ctx, body); err != nil {
				return errors.Errorf("PreWriteResponsePlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}

func (p *pluginContainer) doPostWriteResponse(ctx *Context, body interface{}) error {
	for _, _plugin := range p.plugins {
		if plugin, ok := _plugin.(PostWriteResponsePlugin); ok {
			if err := plugin.PostWriteResponse(ctx, body); err != nil {
				return errors.Errorf("PostWriteResponsePlugin(%s): %s", plugin.Name(), err.Error())
			}
		}
	}
	return nil
}
