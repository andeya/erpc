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

package heartbeat

import (
	"net/url"
	"strconv"
	"time"

	"github.com/henrylee2cn/goutil/coarsetime"
	tp "github.com/henrylee2cn/teleport"
)

// NewPong returns a heartbeat receiver plugin.
func NewPong() Pong {
	return new(heartPong)
}

type (
	// Pong receive heartbeat.
	Pong interface {
		// Name returns name.
		Name() string
		// PostNewPeer runs ping woker.
		PostNewPeer(peer tp.EarlyPeer) error
		// PostWriteCall updates heartbeat information.
		PostWriteCall(ctx tp.WriteCtx) *tp.Rerror
		// PostWritePush updates heartbeat information.
		PostWritePush(ctx tp.WriteCtx) *tp.Rerror
		// PostReadCallHeader updates heartbeat information.
		PostReadCallHeader(ctx tp.ReadCtx) *tp.Rerror
		// PostReadPushHeader updates heartbeat information.
		PostReadPushHeader(ctx tp.ReadCtx) *tp.Rerror
	}
	heartPong struct{}
)

var (
	_ tp.PostNewPeerPlugin        = Pong(nil)
	_ tp.PostWriteCallPlugin      = Pong(nil)
	_ tp.PostWritePushPlugin      = Pong(nil)
	_ tp.PostReadCallHeaderPlugin = Pong(nil)
	_ tp.PostReadPushHeaderPlugin = Pong(nil)
)

func (h *heartPong) Name() string {
	return "heart-pong"
}

func (h *heartPong) PostNewPeer(peer tp.EarlyPeer) error {
	peer.RouteCallFunc((*pongCall).heartbeat)
	peer.RoutePushFunc((*pongPush).heartbeat)
	rangeSession := peer.RangeSession
	const initial = time.Second*minRateSecond - 1
	interval := initial
	go func() {
		for {
			time.Sleep(interval)
			rangeSession(func(sess tp.Session) bool {
				info, ok := getHeartbeatInfo(sess.Swap())
				if !ok {
					return true
				}
				cp := info.elemCopy()
				if sess.Health() && cp.last.Add(cp.rate*2).Before(coarsetime.CeilingTimeNow()) {
					sess.Close()
				}
				if cp.rate < interval || interval == initial {
					interval = cp.rate
				}
				return true
			})
		}
	}()
	return nil
}

func (h *heartPong) PostReadCallHeader(ctx tp.ReadCtx) *tp.Rerror {
	h.update(ctx)
	return nil
}

func (h *heartPong) PostReadPushHeader(ctx tp.ReadCtx) *tp.Rerror {
	h.update(ctx)
	return nil
}

func (h *heartPong) PostWriteCall(ctx tp.WriteCtx) *tp.Rerror {
	return h.PostWritePush(ctx)
}

func (h *heartPong) PostWritePush(ctx tp.WriteCtx) *tp.Rerror {
	sess := ctx.Session()
	if !sess.Health() {
		return nil
	}
	updateHeartbeatInfo(sess.Swap(), 0)
	return nil
}

func (h *heartPong) update(ctx tp.ReadCtx) {
	if ctx.Path() == HeartbeatUri {
		return
	}
	sess := ctx.Session()
	if !sess.Health() {
		return
	}
	updateHeartbeatInfo(sess.Swap(), 0)
}

type pongCall struct {
	tp.CallCtx
}

func (ctx *pongCall) heartbeat(_ *struct{}) (*struct{}, *tp.Rerror) {
	return nil, handelHeartbeat(ctx.Session(), ctx.Query())
}

type pongPush struct {
	tp.PushCtx
}

func (ctx *pongPush) heartbeat(_ *struct{}) *tp.Rerror {
	return handelHeartbeat(ctx.Session(), ctx.Query())
}

func handelHeartbeat(sess tp.Session, query url.Values) *tp.Rerror {
	rateStr := query.Get(heartbeatQueryKey)
	rateSecond := parseHeartbeatRateSecond(rateStr)
	isFirst := updateHeartbeatInfo(sess.Swap(), time.Second*time.Duration(rateSecond))
	if isFirst && rateSecond == -1 {
		return tp.NewRerror(tp.CodeBadPacket, "Invalid Heartbeat Rate", rateStr)
	}
	if rateSecond == 0 {
		tp.Tracef("heart-pong: %s", sess.Id())
	} else {
		tp.Tracef("heart-pong: %s, set rate: %ds", sess.Id(), rateSecond)
	}
	return nil
}

func parseHeartbeatRateSecond(s string) int {
	if len(s) == 0 {
		return 0
	}
	r, err := strconv.Atoi(s)
	if err != nil || r < 0 {
		return -1
	}
	return r
}
