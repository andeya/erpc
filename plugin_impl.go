package erpc

import "net"

var (
	_ Plugin                    = (*PluginImpl)(nil)
	_ PreNewPeerPlugin          = (*PluginImpl)(nil)
	_ PostNewPeerPlugin         = (*PluginImpl)(nil)
	_ PostRegPlugin             = (*PluginImpl)(nil)
	_ PostListenPlugin          = (*PluginImpl)(nil)
	_ PreDialPlugin             = (*PluginImpl)(nil)
	_ PostDialPlugin            = (*PluginImpl)(nil)
	_ PostAcceptPlugin          = (*PluginImpl)(nil)
	_ PreWriteCallPlugin        = (*PluginImpl)(nil)
	_ PostWriteCallPlugin       = (*PluginImpl)(nil)
	_ PreWriteReplyPlugin       = (*PluginImpl)(nil)
	_ PostWriteReplyPlugin      = (*PluginImpl)(nil)
	_ PreWritePushPlugin        = (*PluginImpl)(nil)
	_ PostWritePushPlugin       = (*PluginImpl)(nil)
	_ PreReadHeaderPlugin       = (*PluginImpl)(nil)
	_ PostReadCallHeaderPlugin  = (*PluginImpl)(nil)
	_ PreReadCallBodyPlugin     = (*PluginImpl)(nil)
	_ PostReadCallBodyPlugin    = (*PluginImpl)(nil)
	_ PostReadPushHeaderPlugin  = (*PluginImpl)(nil)
	_ PreReadPushBodyPlugin     = (*PluginImpl)(nil)
	_ PostReadPushBodyPlugin    = (*PluginImpl)(nil)
	_ PostReadReplyHeaderPlugin = (*PluginImpl)(nil)
	_ PreReadReplyBodyPlugin    = (*PluginImpl)(nil)
	_ PostReadReplyBodyPlugin   = (*PluginImpl)(nil)
	_ PostDisconnectPlugin      = (*PluginImpl)(nil)
)

// PluginImpl implemented all plug-in interfaces.
type PluginImpl struct {
	// PluginName is required field
	PluginName string
	// OnPreNewPeer is called before a new peer is created.
	OnPreNewPeer func(*PeerConfig, *PluginContainer) error
	// OnPostNewPeer is called after a new peer is created.
	OnPostNewPeer func(EarlyPeer) error
	// OnPostReg is called after a handler is registered.
	OnPostReg func(*Handler) error
	// OnPostListen is called after a listener is created.
	OnPostListen func(net.Addr) error
	// OnPreDial is called before a dial is created.
	OnPreDial func(localAddr net.Addr, remoteAddr string) *Status
	// OnPostDial is called after a dial is created.
	OnPostDial func(sess PreSession, isRedial bool) *Status
	// OnPostAccept is called after a session is accepted.
	OnPostAccept func(PreSession) *Status
	// OnPreWriteCall is called before a call is written.
	OnPreWriteCall func(WriteCtx) *Status
	// OnPostWriteCall is called after a call is written.
	OnPostWriteCall func(WriteCtx) *Status
	// OnPreWriteReply is called before a reply is written.
	OnPreWriteReply func(WriteCtx) *Status
	// OnPostWriteReply is called after a reply is written.
	OnPostWriteReply func(WriteCtx) *Status
	// OnPreWritePush is called before a push is written.
	OnPreWritePush func(WriteCtx) *Status
	// OnPostWritePush is called after a push is written.
	OnPostWritePush func(WriteCtx) *Status
	// OnPreReadHeader is called before a header is read.
	OnPreReadHeader func(PreCtx) error
	// OnPostReadCallHeader is called after a call header is read.
	OnPostReadCallHeader func(ReadCtx) *Status
	// OnPreReadCallBody is called before a call body is read.
	OnPreReadCallBody func(ReadCtx) *Status
	// OnPostReadCallBody is called after a call body is read.
	OnPostReadCallBody func(ReadCtx) *Status
	// OnPostReadPushHeader is called after a push header is read.
	OnPostReadPushHeader func(ReadCtx) *Status
	// OnPreReadPushBody is called before a push body is read.
	OnPreReadPushBody func(ReadCtx) *Status
	// OnPostReadPushBody is called after a push body is read.
	OnPostReadPushBody func(ReadCtx) *Status
	// OnPostReadReplyHeader is called after a reply header is read.
	OnPostReadReplyHeader func(ReadCtx) *Status
	// OnPreReadReplyBody is called before a reply body is read.
	OnPreReadReplyBody func(ReadCtx) *Status
	// OnPostReadReplyBody is called after a reply body is read.
	OnPostReadReplyBody func(ReadCtx) *Status
	// OnPostDisconnect is called after a session is disconnected.
	OnPostDisconnect func(BaseSession) *Status
}

// Name returns the name of the plugin.
func (p *PluginImpl) Name() string {
	return p.PluginName
}

// PreNewPeer is called before a new peer is created.
func (p *PluginImpl) PreNewPeer(peerConfig *PeerConfig, pluginContainer *PluginContainer) error {
	if p.OnPreNewPeer == nil {
		return nil
	}
	return p.OnPreNewPeer(peerConfig, pluginContainer)
}

// PostNewPeer is called after a new peer is created.
func (p *PluginImpl) PostNewPeer(earlyPeer EarlyPeer) error {
	if p.OnPostNewPeer == nil {
		return nil
	}
	return p.OnPostNewPeer(earlyPeer)
}

// PostReg is called after a handler is registered.
func (p *PluginImpl) PostReg(handler *Handler) error {
	if p.OnPostReg == nil {
		return nil
	}
	return p.OnPostReg(handler)
}

// PostListen is called after a listener is created.
func (p *PluginImpl) PostListen(addr net.Addr) error {
	if p.OnPostListen == nil {
		return nil
	}
	return p.OnPostListen(addr)
}

// PreDial is called before a dial is created.
func (p *PluginImpl) PreDial(localAddr net.Addr, remoteAddr string) *Status {
	if p.OnPreDial == nil {
		return nil
	}
	return p.OnPreDial(localAddr, remoteAddr)
}

// PostDial is called after a dial is created.
func (p *PluginImpl) PostDial(sess PreSession, isRedial bool) *Status {
	if p.OnPostDial == nil {
		return nil
	}
	return p.OnPostDial(sess, isRedial)
}

// PostAccept is called after a session is accepted.
func (p *PluginImpl) PostAccept(sess PreSession) *Status {
	if p.OnPostAccept == nil {
		return nil
	}
	return p.OnPostAccept(sess)
}

// PreWriteCall is called before a call is written.
func (p *PluginImpl) PreWriteCall(writeCtx WriteCtx) *Status {
	if p.OnPreWriteCall == nil {
		return nil
	}
	return p.OnPreWriteCall(writeCtx)
}

// PostWriteCall is called after a call is written.
func (p *PluginImpl) PostWriteCall(writeCtx WriteCtx) *Status {
	if p.OnPostWriteCall == nil {
		return nil
	}
	return p.OnPostWriteCall(writeCtx)
}

// PreWriteReply is called before a reply is written.
func (p *PluginImpl) PreWriteReply(writeCtx WriteCtx) *Status {
	if p.OnPreWriteReply == nil {
		return nil
	}
	return p.OnPreWriteReply(writeCtx)
}

// PostWriteReply is called after a reply is written.
func (p *PluginImpl) PostWriteReply(writeCtx WriteCtx) *Status {
	if p.OnPostWriteReply == nil {
		return nil
	}
	return p.OnPostWriteReply(writeCtx)
}

// PreWritePush is called before a push is written.
func (p *PluginImpl) PreWritePush(writeCtx WriteCtx) *Status {
	if p.OnPreWritePush == nil {
		return nil
	}
	return p.OnPreWritePush(writeCtx)
}

// PostWritePush is called after a push is written.
func (p *PluginImpl) PostWritePush(writeCtx WriteCtx) *Status {
	if p.OnPostWritePush == nil {
		return nil
	}
	return p.OnPostWritePush(writeCtx)
}

// PreReadHeader is called before a header is read.
func (p *PluginImpl) PreReadHeader(preCtx PreCtx) error {
	if p.OnPreReadHeader == nil {
		return nil
	}
	return p.OnPreReadHeader(preCtx)
}

// PostReadCallHeader is called after a call header is read.
func (p *PluginImpl) PostReadCallHeader(readCtx ReadCtx) *Status {
	if p.OnPostReadCallHeader == nil {
		return nil
	}
	return p.OnPostReadCallHeader(readCtx)
}

// PreReadCallBody is called before a call body is read.
func (p *PluginImpl) PreReadCallBody(readCtx ReadCtx) *Status {
	if p.OnPreReadCallBody == nil {
		return nil
	}
	return p.OnPreReadCallBody(readCtx)
}

// PostReadCallBody is called after a call body is read.
func (p *PluginImpl) PostReadCallBody(readCtx ReadCtx) *Status {
	if p.OnPostReadCallBody == nil {
		return nil
	}
	return p.OnPostReadCallBody(readCtx)
}

// PostReadPushHeader is called after a push header is read.
func (p *PluginImpl) PostReadPushHeader(readCtx ReadCtx) *Status {
	if p.OnPostReadPushHeader == nil {
		return nil
	}
	return p.OnPostReadPushHeader(readCtx)
}

// PreReadPushBody is called before a push body is read.
func (p *PluginImpl) PreReadPushBody(readCtx ReadCtx) *Status {
	if p.OnPreReadPushBody == nil {
		return nil
	}
	return p.OnPreReadPushBody(readCtx)
}

// PostReadPushBody is called after a push body is read.
func (p *PluginImpl) PostReadPushBody(readCtx ReadCtx) *Status {
	if p.OnPostReadPushBody == nil {
		return nil
	}
	return p.OnPostReadPushBody(readCtx)
}

// PostReadReplyHeader is called after a reply header is read.
func (p *PluginImpl) PostReadReplyHeader(readCtx ReadCtx) *Status {
	if p.OnPostReadReplyHeader == nil {
		return nil
	}
	return p.OnPostReadReplyHeader(readCtx)
}

// PreReadReplyBody is called before a reply body is read.
func (p *PluginImpl) PreReadReplyBody(readCtx ReadCtx) *Status {
	if p.OnPreReadReplyBody == nil {
		return nil
	}
	return p.OnPreReadReplyBody(readCtx)
}

// PostReadReplyBody is called after a reply body is read.
func (p *PluginImpl) PostReadReplyBody(readCtx ReadCtx) *Status {
	if p.OnPostReadReplyBody == nil {
		return nil
	}
	return p.OnPostReadReplyBody(readCtx)
}

// PostDisconnect is called after a session is disconnected.
func (p *PluginImpl) PostDisconnect(sess BaseSession) *Status {
	if p.OnPostDisconnect == nil {
		return nil
	}
	return p.OnPostDisconnect(sess)
}
