// Copyright 2015-2018 HenryLee. All Rights Reserved.
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

	"github.com/henrylee2cn/goutil"
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
		PostListen(net.Addr) error
	}
	// PostDialPlugin is executed after dialing.
	PostDialPlugin interface {
		Plugin
		PostDial(PreSession) *Status
	}
	// PostAcceptPlugin is executed after accepting connection.
	PostAcceptPlugin interface {
		Plugin
		PostAccept(PreSession) *Status
	}
	// PreWriteCallPlugin is executed before writing CALL message.
	PreWriteCallPlugin interface {
		Plugin
		PreWriteCall(WriteCtx) *Status
	}
	// PostWriteCallPlugin is executed after successful writing CALL message.
	PostWriteCallPlugin interface {
		Plugin
		PostWriteCall(WriteCtx) *Status
	}
	// PreWriteReplyPlugin is executed before writing REPLY message.
	PreWriteReplyPlugin interface {
		Plugin
		PreWriteReply(WriteCtx) *Status
	}
	// PostWriteReplyPlugin is executed after successful writing REPLY message.
	PostWriteReplyPlugin interface {
		Plugin
		PostWriteReply(WriteCtx) *Status
	}
	// PreWritePushPlugin is executed before writing PUSH message.
	PreWritePushPlugin interface {
		Plugin
		PreWritePush(WriteCtx) *Status
	}
	// PostWritePushPlugin is executed after successful writing PUSH message.
	PostWritePushPlugin interface {
		Plugin
		PostWritePush(WriteCtx) *Status
	}
	// PreReadHeaderPlugin is executed before reading message header.
	PreReadHeaderPlugin interface {
		Plugin
		PreReadHeader(PreCtx) error
	}
	// PostReadCallHeaderPlugin is executed after reading CALL message header.
	PostReadCallHeaderPlugin interface {
		Plugin
		PostReadCallHeader(ReadCtx) *Status
	}
	// PreReadCallBodyPlugin is executed before reading CALL message body.
	PreReadCallBodyPlugin interface {
		Plugin
		PreReadCallBody(ReadCtx) *Status
	}
	// PostReadCallBodyPlugin is executed after reading CALL message body.
	PostReadCallBodyPlugin interface {
		Plugin
		PostReadCallBody(ReadCtx) *Status
	}
	// PostReadPushHeaderPlugin is executed after reading PUSH message header.
	PostReadPushHeaderPlugin interface {
		Plugin
		PostReadPushHeader(ReadCtx) *Status
	}
	// PreReadPushBodyPlugin is executed before reading PUSH message body.
	PreReadPushBodyPlugin interface {
		Plugin
		PreReadPushBody(ReadCtx) *Status
	}
	// PostReadPushBodyPlugin is executed after reading PUSH message body.
	PostReadPushBodyPlugin interface {
		Plugin
		PostReadPushBody(ReadCtx) *Status
	}
	// PostReadReplyHeaderPlugin is executed after reading REPLY message header.
	PostReadReplyHeaderPlugin interface {
		Plugin
		PostReadReplyHeader(ReadCtx) *Status
	}
	// PreReadReplyBodyPlugin is executed before reading REPLY message body.
	PreReadReplyBodyPlugin interface {
		Plugin
		PreReadReplyBody(ReadCtx) *Status
	}
	// PostReadReplyBodyPlugin is executed after reading REPLY message body.
	PostReadReplyBodyPlugin interface {
		Plugin
		PostReadReplyBody(ReadCtx) *Status
	}
	// PostDisconnectPlugin is executed after disconnection.
	PostDisconnectPlugin interface {
		Plugin
		PostDisconnect(BaseSession) *Status
	}
)

// PluginContainer a plugin container
type PluginContainer struct {
	*pluginSingleContainer
	left        *pluginSingleContainer
	middle      *pluginSingleContainer
	right       *pluginSingleContainer
	refreshTree func()
}

// newPluginContainer new a plugin container.
func newPluginContainer() *PluginContainer {
	p := &PluginContainer{
		pluginSingleContainer: newPluginSingleContainer(),
		left:                  newPluginSingleContainer(),
		middle:                newPluginSingleContainer(),
		right:                 newPluginSingleContainer(),
	}
	p.refreshTree = func() { p.refresh() }
	return p
}

func (p *PluginContainer) cloneAndAppendMiddle(plugins ...Plugin) *PluginContainer {
	middle := newPluginSingleContainer()
	middle.plugins = append(p.middle.GetAll(), plugins...)

	newPluginContainer := newPluginContainer()
	newPluginContainer.middle = middle
	newPluginContainer.left = p.left
	newPluginContainer.right = p.right
	newPluginContainer.refresh()

	oldRefreshTree := p.refreshTree
	p.refreshTree = func() {
		oldRefreshTree()
		newPluginContainer.refresh()
	}
	return newPluginContainer
}

// AppendLeft appends plugins on the left side of the pluginContainer.
func (p *PluginContainer) AppendLeft(plugins ...Plugin) {
	p.left.appendLeft(plugins...)
	p.refreshTree()
}

// AppendRight appends plugins on the right side of the pluginContainer.
func (p *PluginContainer) AppendRight(plugins ...Plugin) {
	p.right.appendRight(plugins...)
	p.refreshTree()
}

// Remove removes a plugin by it's name.
func (p *PluginContainer) Remove(pluginName string) error {
	err := p.pluginSingleContainer.remove(pluginName)
	if err != nil {
		return err
	}
	p.left.remove(pluginName)
	p.middle.remove(pluginName)
	p.right.remove(pluginName)
	p.refreshTree()
	return nil
}

func (p *PluginContainer) refresh() {
	count := len(p.left.plugins) + len(p.middle.plugins) + len(p.right.plugins)
	allPlugins := make([]Plugin, count)
	copy(allPlugins[0:], p.left.plugins)
	copy(allPlugins[0+len(p.left.plugins):], p.middle.plugins)
	copy(allPlugins[0+len(p.left.plugins)+len(p.middle.plugins):], p.right.plugins)
	m := make(map[string]bool, count)
	for _, plugin := range allPlugins {
		if plugin == nil {
			Fatalf("plugin cannot be nil!")
			return
		}
		if m[plugin.Name()] {
			Fatalf("repeat add plugin: %s", plugin.Name())
			return
		}
		m[plugin.Name()] = true
	}
	p.pluginSingleContainer.plugins = allPlugins
}

// pluginSingleContainer plugins container.
type pluginSingleContainer struct {
	plugins []Plugin
}

// newPluginSingleContainer new a plugin container.
func newPluginSingleContainer() *pluginSingleContainer {
	return &pluginSingleContainer{
		plugins: make([]Plugin, 0),
	}
}

// appendLeft appends plugins on the left side of the pluginContainer.
func (p *pluginSingleContainer) appendLeft(plugins ...Plugin) {
	if len(plugins) == 0 {
		return
	}
	p.plugins = append(plugins, p.plugins...)
}

// appendRight appends plugins on the right side of the pluginContainer.
func (p *pluginSingleContainer) appendRight(plugins ...Plugin) {
	if len(plugins) == 0 {
		return
	}
	p.plugins = append(p.plugins, plugins...)
}

// GetByName returns a plugin instance by it's name.
func (p *pluginSingleContainer) GetByName(pluginName string) Plugin {
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
func (p *pluginSingleContainer) GetAll() []Plugin {
	return p.plugins
}

// remove removes a plugin by it's name.
func (p *pluginSingleContainer) remove(pluginName string) error {
	if p.plugins == nil {
		return errors.New("no plugins are registered yet")
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
		return errors.New("cannot remove a plugin which isn't exists")
	}
	p.plugins = append(p.plugins[:indexToRemove], p.plugins[indexToRemove+1:]...)
	return nil
}

// PreNewPeer executes the defined plugins before creating peer.
func (p *PluginContainer) preNewPeer(peerConfig *PeerConfig) {
	var err error
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreNewPeerPlugin); ok {
			if err = _plugin.PreNewPeer(peerConfig, p); err != nil {
				Fatalf("[PreNewPeerPlugin:%s] %s", plugin.Name(), err.Error())
				return
			}
		}
	}
}

// PostNewPeer executes the defined plugins after creating peer.
func (p *pluginSingleContainer) postNewPeer(peer EarlyPeer) {
	var err error
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostNewPeerPlugin); ok {
			if err = _plugin.PostNewPeer(peer); err != nil {
				Fatalf("[PostNewPeerPlugin:%s] %s", plugin.Name(), err.Error())
				return
			}
		}
	}
}

// PostReg executes the defined plugins before registering handler.
func (p *pluginSingleContainer) postReg(h *Handler) {
	var err error
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostRegPlugin); ok {
			if err = _plugin.PostReg(h); err != nil {
				Fatalf("[PostRegPlugin:%s] register handler:%s %s, error:%s", plugin.Name(), h.RouterTypeName(), h.Name(), err.Error())
				return
			}
		}
	}
}

// PostListen is executed between listening and accepting.
func (p *pluginSingleContainer) postListen(addr net.Addr) {
	var err error
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostListenPlugin); ok {
			if err = _plugin.PostListen(addr); err != nil {
				Fatalf("[PostListenPlugin:%s] network:%s, addr:%s, error:%s", plugin.Name(), addr.Network(), addr.String(), err.Error())
				return
			}
		}
	}
	return
}

// PostDial executes the defined plugins after dialing.
func (p *pluginSingleContainer) postDial(sess PreSession) (stat *Status) {
	var pluginName string
	defer func() {
		if p := recover(); p != nil {
			Errorf("[PostDialPlugin:%s] network:%s, addr:%s, panic:%v\n%s", pluginName, sess.RemoteAddr().Network(), sess.RemoteAddr().String(), p, goutil.PanicTrace(2))
			stat = statDialFailed.Copy(p)
		}
	}()
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostDialPlugin); ok {
			pluginName = plugin.Name()
			if stat = _plugin.PostDial(sess); !stat.OK() {
				Debugf("[PostDialPlugin:%s] network:%s, addr:%s, error:%s", pluginName, sess.RemoteAddr().Network(), sess.RemoteAddr().String(), stat.String())
				return stat
			}
		}
	}
	return nil
}

// PostAccept executes the defined plugins after accepting connection.
func (p *pluginSingleContainer) postAccept(sess PreSession) (stat *Status) {
	var pluginName string
	defer func() {
		if p := recover(); p != nil {
			Errorf("[PostAcceptPlugin:%s] network:%s, addr:%s, panic:%v\n%s", pluginName, sess.RemoteAddr().Network(), sess.RemoteAddr().String(), p, goutil.PanicTrace(2))
			stat = statInternalServerError.Copy(p)
		}
	}()
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostAcceptPlugin); ok {
			pluginName = plugin.Name()
			if stat = _plugin.PostAccept(sess); !stat.OK() {
				Debugf("[PostAcceptPlugin:%s] network:%s, addr:%s, error:%s", pluginName, sess.RemoteAddr().Network(), sess.RemoteAddr().String(), stat.String())
				return stat
			}
		}
	}
	return nil
}

// PreWriteCall executes the defined plugins before writing CALL message.
func (p *pluginSingleContainer) preWriteCall(ctx WriteCtx) *Status {
	var stat *Status
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreWriteCallPlugin); ok {
			if stat = _plugin.PreWriteCall(ctx); !stat.OK() {
				Debugf("[PreWriteCallPlugin:%s] %s", plugin.Name(), stat.String())
				return stat
			}
		}
	}
	return nil
}

// PostWriteCall executes the defined plugins after successful writing CALL message.
func (p *pluginSingleContainer) postWriteCall(ctx WriteCtx) *Status {
	var stat *Status
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostWriteCallPlugin); ok {
			if stat = _plugin.PostWriteCall(ctx); !stat.OK() {
				Errorf("[PostWriteCallPlugin:%s] %s", plugin.Name(), stat.String())
				return stat
			}
		}
	}
	return nil
}

// PreWriteReply executes the defined plugins before writing REPLY message.
func (p *pluginSingleContainer) preWriteReply(ctx WriteCtx) {
	var stat *Status
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreWriteReplyPlugin); ok {
			if stat = _plugin.PreWriteReply(ctx); !stat.OK() {
				Errorf("[PreWriteReplyPlugin:%s] %s", plugin.Name(), stat.String())
				return
			}
		}
	}
}

// PostWriteReply executes the defined plugins after successful writing REPLY message.
func (p *pluginSingleContainer) postWriteReply(ctx WriteCtx) {
	var stat *Status
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostWriteReplyPlugin); ok {
			if stat = _plugin.PostWriteReply(ctx); !stat.OK() {
				Errorf("[PostWriteReplyPlugin:%s] %s", plugin.Name(), stat.String())
				return
			}
		}
	}
}

// PreWritePush executes the defined plugins before writing PUSH message.
func (p *pluginSingleContainer) preWritePush(ctx WriteCtx) *Status {
	var stat *Status
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreWritePushPlugin); ok {
			if stat = _plugin.PreWritePush(ctx); !stat.OK() {
				Debugf("[PreWritePushPlugin:%s] %s", plugin.Name(), stat.String())
				return stat
			}
		}
	}
	return nil
}

// PostWritePush executes the defined plugins after successful writing PUSH message.
func (p *pluginSingleContainer) postWritePush(ctx WriteCtx) *Status {
	var stat *Status
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostWritePushPlugin); ok {
			if stat = _plugin.PostWritePush(ctx); !stat.OK() {
				Errorf("[PostWritePushPlugin:%s] %s", plugin.Name(), stat.String())
				return stat
			}
		}
	}
	return nil
}

// PreReadHeader executes the defined plugins before reading message header.
func (p *pluginSingleContainer) preReadHeader(ctx PreCtx) error {
	var err error
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadHeaderPlugin); ok {
			if err = _plugin.PreReadHeader(ctx); err != nil {
				Debugf("[PreReadHeaderPlugin:%s] disconnected when reading: %s", plugin.Name(), err.Error())
				return err
			}
		}
	}
	return nil
}

// PostReadCallHeader executes the defined plugins after reading CALL message header.
func (p *pluginSingleContainer) postReadCallHeader(ctx ReadCtx) *Status {
	var stat *Status
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadCallHeaderPlugin); ok {
			if stat = _plugin.PostReadCallHeader(ctx); !stat.OK() {
				Errorf("[PostReadCallHeaderPlugin:%s] %s", plugin.Name(), stat.String())
				return stat
			}
		}
	}
	return nil
}

// PreReadCallBody executes the defined plugins before reading CALL message body.
func (p *pluginSingleContainer) preReadCallBody(ctx ReadCtx) *Status {
	var stat *Status
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadCallBodyPlugin); ok {
			if stat = _plugin.PreReadCallBody(ctx); !stat.OK() {
				Errorf("[PreReadCallBodyPlugin:%s] %s", plugin.Name(), stat.String())
				return stat
			}
		}
	}
	return nil
}

// PostReadCallBody executes the defined plugins after reading CALL message body.
func (p *pluginSingleContainer) postReadCallBody(ctx ReadCtx) *Status {
	var stat *Status
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadCallBodyPlugin); ok {
			if stat = _plugin.PostReadCallBody(ctx); !stat.OK() {
				Errorf("[PostReadCallBodyPlugin:%s] %s", plugin.Name(), stat.String())
				return stat
			}
		}
	}
	return nil
}

// PostReadPushHeader executes the defined plugins after reading PUSH message header.
func (p *pluginSingleContainer) postReadPushHeader(ctx ReadCtx) *Status {
	var stat *Status
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadPushHeaderPlugin); ok {
			if stat = _plugin.PostReadPushHeader(ctx); !stat.OK() {
				Errorf("[PostReadPushHeaderPlugin:%s] %s", plugin.Name(), stat.String())
				return stat
			}
		}
	}
	return nil
}

// PreReadPushBody executes the defined plugins before reading PUSH message body.
func (p *pluginSingleContainer) preReadPushBody(ctx ReadCtx) *Status {
	var stat *Status
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadPushBodyPlugin); ok {
			if stat = _plugin.PreReadPushBody(ctx); !stat.OK() {
				Errorf("[PreReadPushBodyPlugin:%s] %s", plugin.Name(), stat.String())
				return stat
			}
		}
	}
	return nil
}

// PostReadPushBody executes the defined plugins after reading PUSH message body.
func (p *pluginSingleContainer) postReadPushBody(ctx ReadCtx) *Status {
	var stat *Status
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadPushBodyPlugin); ok {
			if stat = _plugin.PostReadPushBody(ctx); !stat.OK() {
				Errorf("[PostReadPushBodyPlugin:%s] %s", plugin.Name(), stat.String())
				return stat
			}
		}
	}
	return nil
}

// PostReadReplyHeader executes the defined plugins after reading REPLY message header.
func (p *pluginSingleContainer) postReadReplyHeader(ctx ReadCtx) *Status {
	var stat *Status
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadReplyHeaderPlugin); ok {
			if stat = _plugin.PostReadReplyHeader(ctx); !stat.OK() {
				Errorf("[PostReadReplyHeaderPlugin:%s] %s", plugin.Name(), stat.String())
				return stat
			}
		}
	}
	return nil
}

// PreReadReplyBody executes the defined plugins before reading REPLY message body.
func (p *pluginSingleContainer) preReadReplyBody(ctx ReadCtx) *Status {
	var stat *Status
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PreReadReplyBodyPlugin); ok {
			if stat = _plugin.PreReadReplyBody(ctx); !stat.OK() {
				Errorf("[PreReadReplyBodyPlugin:%s] %s", plugin.Name(), stat.String())
				return stat
			}
		}
	}
	return nil
}

// PostReadReplyBody executes the defined plugins after reading REPLY message body.
func (p *pluginSingleContainer) postReadReplyBody(ctx ReadCtx) *Status {
	var stat *Status
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostReadReplyBodyPlugin); ok {
			if stat = _plugin.PostReadReplyBody(ctx); !stat.OK() {
				Errorf("[PostReadReplyBodyPlugin:%s] %s", plugin.Name(), stat.String())
				return stat
			}
		}
	}
	return nil
}

// PostDisconnect executes the defined plugins after disconnection.
func (p *pluginSingleContainer) postDisconnect(sess BaseSession) *Status {
	var stat *Status
	for _, plugin := range p.plugins {
		if _plugin, ok := plugin.(PostDisconnectPlugin); ok {
			if stat = _plugin.PostDisconnect(sess); !stat.OK() {
				Errorf("[PostDisconnectPlugin:%s] %s", plugin.Name(), stat.String())
				return stat
			}
		}
	}
	return nil
}

func warnInvaildHandlerHooks(plugin []Plugin) {
	for _, p := range plugin {
		switch p.(type) {
		case PreNewPeerPlugin:
			Debugf("invalid PreNewPeerPlugin in router: %s", p.Name())
		case PostNewPeerPlugin:
			Debugf("invalid PostNewPeerPlugin in router: %s", p.Name())
		case PostDialPlugin:
			Debugf("invalid PostDialPlugin in router: %s", p.Name())
		case PostAcceptPlugin:
			Debugf("invalid PostAcceptPlugin in router: %s", p.Name())
		case PreWriteCallPlugin:
			Debugf("invalid PreWriteCallPlugin in router: %s", p.Name())
		case PostWriteCallPlugin:
			Debugf("invalid PostWriteCallPlugin in router: %s", p.Name())
		case PreWritePushPlugin:
			Debugf("invalid PreWritePushPlugin in router: %s", p.Name())
		case PostWritePushPlugin:
			Debugf("invalid PostWritePushPlugin in router: %s", p.Name())
		case PreReadHeaderPlugin:
			Debugf("invalid PreReadHeaderPlugin in router: %s", p.Name())
		case PostReadCallHeaderPlugin:
			Debugf("invalid PostReadCallHeaderPlugin in router: %s", p.Name())
		case PostReadPushHeaderPlugin:
			Debugf("invalid PostReadPushHeaderPlugin in router: %s", p.Name())
		}
	}
}
