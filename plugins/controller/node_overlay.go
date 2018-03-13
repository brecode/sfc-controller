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

// The core plugin which drives the SFC Controller.  The core initializes the
// CNP dirver plugin based on command line args.  The database is initialized,
// and a resync is preformed based on what was already in the database.

package controller

import (
	"fmt"

	"github.com/ligato/sfc-controller/plugins/controller/idapi"
	"github.com/ligato/sfc-controller/plugins/controller/model"
)

// NodeOverlayValidate sees if the overlay is defined properly
func (s *Plugin) NodeOverlayValidate(no *controller.NodeOverlay) error {

	switch no.NodeOverlayType {
	case controller.NodeOverlayTypeMesh:
	case controller.NodeOverlayTypeHubAndSpoke:
	default:
		return fmt.Errorf("Node overlay: %s overlay type '%s' not recognized",
			no.Name, no.NodeOverlayType)
	}

	switch no.ConnectionType {
	case controller.NodeOverlayConnectionTypeVxlan:
		switch no.NodeOverlayType {
		case controller.NodeOverlayTypeMesh:
			if no.VxlanMeshParms == nil {
				return fmt.Errorf("Node overlay: %s vxlan mesh parameters not specified",
					no.Name)
			}
			if no.VxlanMeshParms.VniRangeStart > no.VxlanMeshParms.VniRangeEnd ||
				no.VxlanMeshParms.VniRangeStart == 0 || no.VxlanMeshParms.VniRangeEnd == 0 {
				return fmt.Errorf("Node overlay: %s vxlan vni range invalid", no.Name)
			}
		case controller.NodeOverlayTypeHubAndSpoke:
			if no.VxlanHubAndSpokeParms == nil {
				return fmt.Errorf("Node overlay: %s vxlan hub and spoke parameters not specified",
					no.Name)
			}
			if no.VxlanHubAndSpokeParms.Vni == 0 {
				return fmt.Errorf("Node overlay: %s vxlan vni invalid", no.Name)
			}
		default:
			return fmt.Errorf("Node overlay: %s overlay type %s not supported for connection type '%s'",
				no.Name, no.NodeOverlayType, no.ConnectionType)
		}
	default:
		return fmt.Errorf("Node overlay: %s connection type '%s' not recognized",
			no.Name, no.ConnectionType)
	}

	return nil
}

// NodeOverlayCreate add to ram cache and run side effects
func (s *Plugin) NodeOverlayCreate(no *controller.NodeOverlay, render bool) error {

	if err := s.NodeOverlayValidate(no); err != nil {
		return err
	}

	s.ramConfigCache.NodeOverlays[no.Name] = *no

	if err := s.NodeOverlayWriteToDatastore(no); err != nil {
		return err
	}

	if no.VxlanMeshParms != nil {
		s.ramConfigCache.NodeOverlayVniAllocators[no.Name] =
			idapi.NewVxlanVniAllocator(no.VxlanMeshParms.VniRangeStart,
				no.VxlanMeshParms.VniRangeEnd)
	}
	// process all services as the node overlay meshing strategy may have changed
	if render {
		if err := s.VNFServicesRender(); err != nil {
			return err
		}
	}

	return nil
}

// NodeOverlaysCreate add each to ram cache and run topology side effects
func (s *Plugin) NodeOverlaysCreate(nodeOverlays []controller.NodeOverlay, render bool) error {

	for _, no := range nodeOverlays {

		if err := s.NodeOverlayCreate(&no, false); err != nil {
			return err
		}
	}

	if render {
		// process all services as the node overlay meshing strategy may have changed
		if err := s.VNFServicesRender(); err != nil {
			return err
		}
	}
	return nil
}

// NodeOverlayAllocateVxlanAddress allocates a free address from the pool
func (s *Plugin) NodeOverlayAllocateVxlanAddress(poolName string, nodeName string) (string, error) {
	
	if vxlanIPAddress, exists := s.ramConfigCache.NodeOverlayVxLanAddresses[nodeName]; exists {
		return vxlanIPAddress, nil
	}
	vxlanIpamPool, err := s.IPAMPoolFindAllocator(poolName, "") // system level pool for vxlans
	if err != nil {
		return "", fmt.Errorf("Cannot find system vxlan pool %s: %s", poolName, err)
	}
	vxlanIPAddress, _, err := vxlanIpamPool.AllocateIPAddress()
	if err != nil {
		return "", fmt.Errorf("Cannot allocate address from node overlay vxlan pool %s", poolName)
	}

	s.ramConfigCache.NodeOverlayVxLanAddresses[nodeName] = vxlanIPAddress

	return vxlanIPAddress, nil
}
