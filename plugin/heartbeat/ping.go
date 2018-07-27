// Heartbeat is a generic timing heartbeat plugin.
//
// Copyright 2018 HenryLee. All Rights Reserved.
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
//
package heartbeat

import (
	"strconv"
	"sync"
	"time"

	"github.com/henrylee2cn/goutil/coarsetime"
	tp "github.com/henrylee2cn/teleport"
)

const (
	// HeartbeatUri heartbeat service URI
	HeartbeatUri      = "/heartbeat"
	heartbeatQueryKey = "hb_"
)

// NewPing returns a heartbeat(CALL or PUSH) sender plugin.
func NewPing(rateSecond int, useCall bool) Ping {
	p := new(heartPing)
	p.useCall = useCall
	p.SetRate(rateSecond)
	return p
}

type (
	// Ping send heartbeat.
	Ping interface {
		// SetRate sets heartbeat rate.
		SetRate(rateSecond int)
		// UseCall uses CALL method to ping.
		UseCall()
		// UsePush uses PUSH method to ping.
		UsePush()
		// Name returns name.
		Name() string
		// PostNewPeer runs ping woker.
		PostNewPeer(peer tp.EarlyPeer) error
		// PostDial initializes heartbeat information.
		PostDial(sess tp.PreSession) *tp.Rerror
		// PostAccept initializes heartbeat information.
		PostAccept(sess tp.PreSession) *tp.Rerror
		// PostWriteCall updates heartbeat information.
		PostWriteCall(ctx tp.WriteCtx) *tp.Rerror
		// PostWritePush updates heartbeat information.
		PostWritePush(ctx tp.WriteCtx) *tp.Rerror
		// PostReadCallHeader updates heartbeat information.
		PostReadCallHeader(ctx tp.ReadCtx) *tp.Rerror
		// PostReadPushHeader updates heartbeat information.
		PostReadPushHeader(ctx tp.ReadCtx) *tp.Rerror
	}
	heartPing struct {
		peer     tp.Peer
		pingRate time.Duration
		uri      string
		useCall  bool
		mu       sync.RWMutex
		once     sync.Once
	}
)

var (
	_ tp.PostNewPeerPlugin        = Ping(nil)
	_ tp.PostDialPlugin           = Ping(nil)
	_ tp.PostAcceptPlugin         = Ping(nil)
	_ tp.PostWriteCallPlugin      = Ping(nil)
	_ tp.PostWritePushPlugin      = Ping(nil)
	_ tp.PostReadCallHeaderPlugin = Ping(nil)
	_ tp.PostReadPushHeaderPlugin = Ping(nil)
)

// SetRate sets heartbeat rate.
func (h *heartPing) SetRate(rateSecond int) {
	if rateSecond < minRateSecond {
		rateSecond = minRateSecond
	}
	h.mu.Lock()
	h.pingRate = time.Second * time.Duration(rateSecond)
	h.uri = HeartbeatUri + "?" + heartbeatQueryKey + "=" + strconv.Itoa(rateSecond)
	h.mu.Unlock()
	tp.Infof("set heartbeat rate: %ds", rateSecond)
}

func (h *heartPing) getRate() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.pingRate
}

func (h *heartPing) getUri() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.uri
}

// UseCall uses CALL method to ping.
func (h *heartPing) UseCall() {
	h.mu.Lock()
	h.useCall = true
	h.mu.Unlock()
}

// UsePush uses PUSH method to ping.
func (h *heartPing) UsePush() {
	h.mu.Lock()
	h.useCall = false
	h.mu.Unlock()
}

func (h *heartPing) isCall() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.useCall
}

// Name returns name.
func (h *heartPing) Name() string {
	return "heart-ping"
}

// PostNewPeer runs ping woker.
func (h *heartPing) PostNewPeer(peer tp.EarlyPeer) error {
	rangeSession := peer.RangeSession
	go func() {
		var isCall bool
		for {
			time.Sleep(h.getRate())
			isCall = h.isCall()
			rangeSession(func(sess tp.Session) bool {
				if !sess.Health() {
					return true
				}
				info, ok := getHeartbeatInfo(sess.Swap())
				if !ok {
					return true
				}
				cp := info.elemCopy()
				if cp.last.Add(cp.rate).After(coarsetime.CeilingTimeNow()) {
					return true
				}
				if isCall {
					h.goCall(sess)
				} else {
					h.goPush(sess)
				}
				return true
			})
		}
	}()
	return nil
}

// PostDial initializes heartbeat information.
func (h *heartPing) PostDial(sess tp.PreSession) *tp.Rerror {
	return h.PostAccept(sess)
}

// PostAccept initializes heartbeat information.
func (h *heartPing) PostAccept(sess tp.PreSession) *tp.Rerror {
	rate := h.getRate()
	initHeartbeatInfo(sess.Swap(), rate)
	return nil
}

// PostWriteCall updates heartbeat information.
func (h *heartPing) PostWriteCall(ctx tp.WriteCtx) *tp.Rerror {
	return h.PostWritePush(ctx)
}

// PostWritePush updates heartbeat information.
func (h *heartPing) PostWritePush(ctx tp.WriteCtx) *tp.Rerror {
	h.update(ctx)
	return nil
}

// PostReadCallHeader updates heartbeat information.
func (h *heartPing) PostReadCallHeader(ctx tp.ReadCtx) *tp.Rerror {
	return h.PostReadPushHeader(ctx)
}

// PostReadPushHeader updates heartbeat information.
func (h *heartPing) PostReadPushHeader(ctx tp.ReadCtx) *tp.Rerror {
	h.update(ctx)
	return nil
}

func (h *heartPing) goCall(sess tp.Session) {
	tp.Go(func() {
		if sess.Call(h.getUri(), nil, nil).Rerror() != nil {
			sess.Close()
		}
	})
}

func (h *heartPing) goPush(sess tp.Session) {
	tp.Go(func() {
		if sess.Push(h.getUri(), nil) != nil {
			sess.Close()
		}
	})
}

func (h *heartPing) update(ctx tp.PreCtx) {
	sess := ctx.Session()
	if !sess.Health() {
		return
	}
	updateHeartbeatInfo(sess.Swap(), h.getRate())
}
