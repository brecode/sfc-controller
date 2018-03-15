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

// VNFServiceMeshValidate sees if the service mesh is defined properly
func (s *Plugin) VNFServiceMeshValidate(vsm *controller.VNFServiceMesh) error {

	switch vsm.ServiceMeshType {
	case controller.VNFServiceMeshTypeMesh:
	case controller.VNFServiceMeshTypeHubAndSpoke:
	default:
		return fmt.Errorf("VNF service mesh: %s service mesh type '%s' not recognized",
			vsm.Name, vsm.ServiceMeshType)
	}

	switch vsm.ConnectionType {
	case controller.VNFServiceMeshConnectionTypeVxlan:
		switch vsm.ServiceMeshType {
		case controller.VNFServiceMeshTypeMesh:
			if vsm.VxlanMeshParms == nil {
				return fmt.Errorf("VNF service mesh: %s vxlan mesh parameters not specified",
					vsm.Name)
			}
			if vsm.VxlanMeshParms.VniRangeStart > vsm.VxlanMeshParms.VniRangeEnd ||
				vsm.VxlanMeshParms.VniRangeStart == 0 || vsm.VxlanMeshParms.VniRangeEnd == 0 {
				return fmt.Errorf("VNF service mesh: %s vxlan vni range invalid", vsm.Name)
			}
		case controller.VNFServiceMeshTypeHubAndSpoke:
			if vsm.VxlanHubAndSpokeParms == nil {
				return fmt.Errorf("VNF service mesh: %s vxlan hub and spoke parameters not specified",
					vsm.Name)
			}
			if vsm.VxlanHubAndSpokeParms.Vni == 0 {
				return fmt.Errorf("VNF service mesh: %s vxlan vni invalid", vsm.Name)
			}
		default:
			return fmt.Errorf("VNF service mesh: %s service mesh type %s not supported for connection type '%s'",
				vsm.Name, vsm.ServiceMeshType, vsm.ConnectionType)
		}
	default:
		return fmt.Errorf("VNF service mesh: %s connection type '%s' not recognized",
			vsm.Name, vsm.ConnectionType)
	}

	return nil
}

// VNFServiceMeshCreate add to ram cache and run side effects
func (s *Plugin) VNFServiceMeshCreate(vsm *controller.VNFServiceMesh, render bool) error {

	if err := s.VNFServiceMeshValidate(vsm); err != nil {
		return err
	}

	s.ramConfigCache.VNFServiceMeshes[vsm.Name] = *vsm

	if err := s.VNFServiceMeshWriteToDatastore(vsm); err != nil {
		return err
	}

	if vsm.VxlanMeshParms != nil {
		s.ramConfigCache.VNFServiceMeshVniAllocators[vsm.Name] =
			idapi.NewVxlanVniAllocator(vsm.VxlanMeshParms.VniRangeStart,
				vsm.VxlanMeshParms.VniRangeEnd)
	}
	// process all services as the VNF service mesh meshing strategy may have changed
	if render {
		if err := s.VNFServicesRender(); err != nil {
			return err
		}
	}

	return nil
}

// VNFServiceMeshesCreate add each to ram cache and run topology side effects
func (s *Plugin) VNFServiceMeshesCreate(VNFServiceMeshes []controller.VNFServiceMesh, render bool) error {

	for _, vsm := range VNFServiceMeshes {

		if err := s.VNFServiceMeshCreate(&vsm, false); err != nil {
			return err
		}
	}

	if render {
		// process all services as the VNF service mesh meshing strategy may have changed
		if err := s.VNFServicesRender(); err != nil {
			return err
		}
	}
	return nil
}

// VNFServiceMeshAllocateVxlanAddress allocates a free address from the pool
func (s *Plugin) VNFServiceMeshAllocateVxlanAddress(poolName string, nodeName string) (string, error) {
	
	if vxlanIPAddress, exists := s.ramConfigCache.VNFServiceMeshVxLanAddresses[nodeName]; exists {
		return vxlanIPAddress, nil
	}
	vxlanIpamPool, err := s.IPAMPoolFindAllocator(poolName, "") // system level pool for vxlans
	if err != nil {
		return "", fmt.Errorf("Cannot find system vxlan pool %s: %s", poolName, err)
	}
	vxlanIPAddress, _, err := vxlanIpamPool.AllocateIPAddress()
	if err != nil {
		return "", fmt.Errorf("Cannot allocate address from VNF service mesh vxlan pool %s", poolName)
	}

	s.ramConfigCache.VNFServiceMeshVxLanAddresses[nodeName] = vxlanIPAddress

	return vxlanIPAddress, nil
}
