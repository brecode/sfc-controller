// Copyright (c) 2017 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	// "fmt"
	// "github.com/ligato/sfc-controller/plugins/controller/model"
	// "github.com/ligato/sfc-controller/plugins/controller/vppagentapi"
)

// ResolveMtu uses this input parm or the system default
func (s *Plugin) ResolveMtu(mtu uint32) uint32 {

	if mtu == 0 {
		mtu = s.ramConfigCache.SysParms.Mtu
	}
	return mtu
}

// ResolveRxMode uses this input parm or the system default
func (s *Plugin) ResolveRxMode(rxMode string) string {

	if rxMode == "" {
		rxMode = s.ramConfigCache.SysParms.RxMode
	}
	return rxMode
}
