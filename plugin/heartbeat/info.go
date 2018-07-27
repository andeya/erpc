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
	"sync"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/coarsetime"
)

type swapKey byte

const (
	heartbeatSwapKey swapKey = 0
	minRateSecond            = 3
)

// heartbeatInfo heartbeat info
type heartbeatInfo struct {
	// heartbeat rate
	rate time.Duration
	// last heartbeat time
	last time.Time
	mu   sync.RWMutex
}

func (h *heartbeatInfo) elemCopy() heartbeatInfo {
	h.mu.RLock()
	copy := heartbeatInfo{
		rate: h.rate,
		last: h.last,
	}
	h.mu.RUnlock()
	return copy
}
func (h *heartbeatInfo) getRate() time.Duration {
	h.mu.RLock()
	rate := h.rate
	h.mu.RUnlock()
	return rate
}

func (h *heartbeatInfo) getLast() time.Time {
	h.mu.RLock()
	last := h.last
	h.mu.RUnlock()
	return last
}

func initHeartbeatInfo(m goutil.Map, rate time.Duration) {
	m.Store(heartbeatSwapKey, &heartbeatInfo{
		rate: rate,
		last: coarsetime.CeilingTimeNow(),
	})
}

// getHeartbeatInfo gets heartbeat info from session.
func getHeartbeatInfo(m goutil.Map) (*heartbeatInfo, bool) {
	_info, ok := m.Load(heartbeatSwapKey)
	if !ok {
		return nil, false
	}
	return _info.(*heartbeatInfo), true
}

// updateHeartbeatInfo updates heartbeat info of session.
func updateHeartbeatInfo(m goutil.Map, rate time.Duration) (isFirst bool) {
	info, ok := getHeartbeatInfo(m)
	if !ok {
		isFirst = true
		if rate > 0 {
			initHeartbeatInfo(m, rate)
		}
		return
	}
	info.mu.Lock()
	if rate > 0 {
		info.rate = rate
	}
	info.last = coarsetime.CeilingTimeNow()
	info.mu.Unlock()
	return
}
