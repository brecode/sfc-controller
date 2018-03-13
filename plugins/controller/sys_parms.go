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
	"fmt"
	//"fmt"

	//"github.com/ligato/sfc-controller/plugins/controller/idapi/ipam"
	"github.com/ligato/sfc-controller/plugins/controller/model"
	//"github.com/ligato/sfc-controller/plugins/controller/vppagentapi"
)

// GetDefaultSystemBDParms return default if NO BDParms or sys template provided
func (s *Plugin) GetDefaultSystemBDParms() *controller.BDParms {
	bdParms := &controller.BDParms{
		Flood:               true,
		UnknownUnicastFlood: true,
		Learn:               true,
		Forward:             true,
		ArpTermination:      false,
		MacAgeMinutes:       0,
	}
	return bdParms
}

// FindL2BDTemplate by name
func (s *Plugin) FindL2BDTemplate(templateName string) *controller.BDParms {
	for _, l2bdt := range s.ramConfigCache.SysParms.L2BdTemplates {
		if templateName == l2bdt.Name {
			return l2bdt
		}
	}
	return nil
}

// SysParmsValidate validates all the fields
func (s *Plugin) SysParmsValidate(sp *controller.SystemParameters) error {

	if sp.Mtu == 0 {
		sp.Mtu = 1500
	}
	if sp.DefaultStaticRoutePreference == 0 {
		sp.DefaultStaticRoutePreference = 5
	}
	if sp.RxMode != "" {
		switch sp.RxMode {
		case controller.RxModeAdaptive:
		case controller.RxModeInterrupt:
		case controller.RxModePolling:
		default:
			return fmt.Errorf("SysParm: Invalid rxMode setting %s", sp.RxMode)
		}
	}
	// for _, ipamPool := range sp.IpamPools {
	// 	if err := ipam.PoolValidate(ipamPool); err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

// SysParmsCreate adds the parameters with appropriate defaults to the ram cache
func (s *Plugin) SysParmsCreate(sp *controller.SystemParameters, render bool) error {

	if err := s.SysParmsValidate(sp); err != nil {
		return err
	}

	if err := s.SysParmsWriteToDatastore(sp); err != nil {
		return err
	}

	s.ramConfigCache.SysParms = *sp

	// create ipam pools
	// for _, ipamPool := range sp.IpamPools {
	// 	s.ramConfigCache.IPAMPoolAllocator[ipamPool.Name] =
	// 		ipam.NewIPAMPoolAllocator(ipamPool)
	// }

	if render {
		if err := s.RenderConfig(); err != nil {
			return err
		}
	}

	return nil
}
