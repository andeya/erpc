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
	"github.com/henrylee2cn/teleport/socket"
)

// PullCmd the command of the pulling operation's response.
type PullCmd struct {
	packet   *socket.Packet
	reply    interface{}
	doneChan chan *PullCmd // Strobes when pull is complete.
}

func (p *PullCmd) done() {
	select {
	case p.doneChan <- p:
		// ok
	default:
		// We don't want to block here. It is the puller's responsibility to make
		// sure the channel has enough buffer space. See comment in GoPull().
		Debugf("teleport: discarding Pull reply due to insufficient result chan capacity")
	}
}
