// Copyright 2017 HenryLee. All Rights Reserved.
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

package plugin

import (
	tp "github.com/henrylee2cn/teleport"
)

// NewRouterRootSetting creates a plugin to set the peer router root.
func NewRouterRootSetting(routerRoot string) tp.PostNewPeerPlugin {
	return &routerRootSetting{routerRoot}
}

type routerRootSetting struct {
	routerRoot string
}

func (r *routerRootSetting) Name() string {
	return "RouterRootSetting"
}

func (r *routerRootSetting) PostNewPeer(peer tp.EarlyPeer) error {
	peer.SetRouterRoot(r.routerRoot)
	return nil
}
