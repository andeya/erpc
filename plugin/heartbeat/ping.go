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

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/goutil/coarsetime"
)

const (
	// HeartbeatServiceMethod heartbeat service method
	HeartbeatServiceMethod = "/heartbeat"
	heartbeatMetaKey       = "hb_"
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
		PostNewPeer(peer erpc.EarlyPeer) error
		// PostDial initializes heartbeat information.
		PostDial(sess erpc.PreSession, isRedial bool) *erpc.Status
		// PostAccept initializes heartbeat information.
		PostAccept(sess erpc.PreSession) *erpc.Status
		// PostWriteCall updates heartbeat information.
		PostWriteCall(ctx erpc.WriteCtx) *erpc.Status
		// PostWritePush updates heartbeat information.
		PostWritePush(ctx erpc.WriteCtx) *erpc.Status
		// PostReadCallHeader updates heartbeat information.
		PostReadCallHeader(ctx erpc.ReadCtx) *erpc.Status
		// PostReadPushHeader updates heartbeat information.
		PostReadPushHeader(ctx erpc.ReadCtx) *erpc.Status
	}
	heartPing struct {
		peer           erpc.Peer
		pingRate       time.Duration
		mu             sync.RWMutex
		once           sync.Once
		pingRateSecond string
		useCall        bool
	}
)

var (
	_ erpc.PostNewPeerPlugin        = Ping(nil)
	_ erpc.PostDialPlugin           = Ping(nil)
	_ erpc.PostAcceptPlugin         = Ping(nil)
	_ erpc.PostWriteCallPlugin      = Ping(nil)
	_ erpc.PostWritePushPlugin      = Ping(nil)
	_ erpc.PostReadCallHeaderPlugin = Ping(nil)
	_ erpc.PostReadPushHeaderPlugin = Ping(nil)
)

// SetRate sets heartbeat rate.
func (h *heartPing) SetRate(rateSecond int) {
	if rateSecond < minRateSecond {
		rateSecond = minRateSecond
	}
	h.mu.Lock()
	h.pingRate = time.Second * time.Duration(rateSecond)
	h.pingRateSecond = strconv.Itoa(rateSecond)
	h.mu.Unlock()
	erpc.Infof("set heartbeat rate: %ds", rateSecond)
}

func (h *heartPing) getRate() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.pingRate
}

func (h *heartPing) getPingRateSecond() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.pingRateSecond
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
func (h *heartPing) PostNewPeer(peer erpc.EarlyPeer) error {
	rangeSession := peer.RangeSession
	go func() {
		var isCall bool
		for {
			time.Sleep(h.getRate())
			isCall = h.isCall()
			rangeSession(func(sess erpc.Session) bool {
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
func (h *heartPing) PostDial(sess erpc.PreSession, _ bool) *erpc.Status {
	return h.PostAccept(sess)
}

// PostAccept initializes heartbeat information.
func (h *heartPing) PostAccept(sess erpc.PreSession) *erpc.Status {
	rate := h.getRate()
	initHeartbeatInfo(sess.Swap(), rate)
	return nil
}

// PostWriteCall updates heartbeat information.
func (h *heartPing) PostWriteCall(ctx erpc.WriteCtx) *erpc.Status {
	return h.PostWritePush(ctx)
}

// PostWritePush updates heartbeat information.
func (h *heartPing) PostWritePush(ctx erpc.WriteCtx) *erpc.Status {
	h.update(ctx)
	return nil
}

// PostReadCallHeader updates heartbeat information.
func (h *heartPing) PostReadCallHeader(ctx erpc.ReadCtx) *erpc.Status {
	return h.PostReadPushHeader(ctx)
}

// PostReadPushHeader updates heartbeat information.
func (h *heartPing) PostReadPushHeader(ctx erpc.ReadCtx) *erpc.Status {
	h.update(ctx)
	return nil
}

func (h *heartPing) goCall(sess erpc.Session) {
	erpc.Go(func() {
		if sess.Call(
			HeartbeatServiceMethod, nil, nil,
			erpc.WithSetMeta(heartbeatMetaKey, h.getPingRateSecond()),
		).Status() != nil {
			sess.Close()
		}
	})
}

func (h *heartPing) goPush(sess erpc.Session) {
	erpc.Go(func() {
		if sess.Push(
			HeartbeatServiceMethod,
			nil,
			erpc.WithSetMeta(heartbeatMetaKey, h.getPingRateSecond()),
		) != nil {
			sess.Close()
		}
	})
}

func (h *heartPing) update(ctx erpc.PreCtx) {
	sess := ctx.Session()
	if !sess.Health() {
		return
	}
	updateHeartbeatInfo(sess.Swap(), h.getRate())
}
