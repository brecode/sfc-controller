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
	"net"
	"github.com/ligato/sfc-controller/plugins/controller/idapi/ipam"
	"github.com/ligato/sfc-controller/plugins/controller/model"
)

// IPAMPoolValidate sees if the overlay is defined properly
func (s *Plugin) IPAMPoolValidate(ipamPool *controller.IPAMPool) error {

	switch ipamPool.Scope {
	case controller.IPAMPoolScopeSystem:
	case controller.IPAMPoolScopeNode:
	case controller.IPAMPoolScopeVNFService:
	default:
		return fmt.Errorf("IPAM pool: %s scope '%s' not recognized",
			ipamPool.Name, ipamPool.Scope)
	}

	_, _, err := net.ParseCIDR(ipamPool.Network)
	//ip, network, err := net.ParseCIDR(ipamPool.Network)
	if err != nil {
		return fmt.Errorf("ipam_pool '%s', %v, expected format i.p.v.4/xx, or ip::v6/xx",
			ipamPool.Name, err)
	}
	// if ipamPool.StartRange != 0 {
	// 	ip = net.ParseIP(ipamPool.StartAddress)
	// 	if ip == nil {
	// 		return fmt.Errorf("ipam_pool '%s', %s not a valid ip address",
	// 			ipamPool.Name, ipamPool.StartAddress)
	// 	}
	// 	if !network.Contains(ip) {
	// 		return fmt.Errorf("ipam_pool '%s', %s not contained in network: %s",
	// 			ipamPool.Name, ipamPool.StartAddress, ipamPool.Network)
	// 	}
	// }
	// if ipamPool.EndRange != 0 {
	// 	ip = net.ParseIP(ipamPool.EndAddress)
	// 	if ip == nil {
	// 		return fmt.Errorf("ipam_pool '%s', %s not a valid ip address",
	// 			ipamPool.Name, ipamPool.EndAddress)
	// 	}
	// 	if !network.Contains(ip) {
	// 		return fmt.Errorf("ipam_pool '%s', %s not contained in network: %s",
	// 			ipamPool.Name, ipamPool.EndAddress, ipamPool.Network)
	// 	}
	// }

	return nil
}

// IPAMPoolCreate add to ram cache and run side effects
func (s *Plugin) IPAMPoolCreate(ipamPool *controller.IPAMPool, render bool) error {

	if err := s.IPAMPoolValidate(ipamPool); err != nil {
		return err
	}

	s.ramConfigCache.IPAMPools[ipamPool.Name] = *ipamPool

	if err := s.IPAMPoolWriteToDatastore(ipamPool); err != nil {
		return err
	}

	switch ipamPool.Scope {
	case controller.IPAMPoolScopeSystem:
		s.IPAMPoolEntityCreate("")
	case controller.IPAMPoolScopeNode:
		for _, n := range s.ramConfigCache.Nodes {
			s.IPAMPoolEntityCreate(n.Name)
		}
	case controller.IPAMPoolScopeVNFService:
		for _, v := range s.ramConfigCache.VNFServices {
			s.IPAMPoolEntityCreate(v.Name)
		}
	}

	// render config based on the ipam pool resource settings
	if render {
		if err := s.RenderConfig(); err != nil {
			return err
		}
	}

	return nil
}

// IPAMPoolFindAllocator returns a scoped allocator for this pool entity
func (s *Plugin) IPAMPoolFindAllocator(poolName string, entityName string) (*ipam.PoolAllocatorType, error) {

	ipamPool, exists := s.ramConfigCache.IPAMPools[poolName]
	if !exists {
		return nil, fmt.Errorf("Cannot find ipam pool %s", poolName)
	}

	allocatorName := contructAllocatorName(&ipamPool, entityName)

	ipamAllocator, exists := s.ramConfigCache.IPAMPoolAllocators[allocatorName]
	if !exists {
			return nil, fmt.Errorf("Cannot find allocator pool %s: allocator: %s",
			poolName, allocatorName)
	}

	return ipamAllocator, nil
}

// IPAMPoolAllocateAddress returns a scoped ip address
func (s *Plugin) IPAMPoolAllocateAddress(poolName string, nodeName string, vsName string) (string, error) {

	ipamPool, exists := s.ramConfigCache.IPAMPools[poolName]
	if !exists {
		return "", fmt.Errorf("Cannot find ipam pool %s", poolName)
	}
	entityName := ""
	switch ipamPool.Scope {
	case controller.IPAMPoolScopeNode:
		entityName = nodeName
	case controller.IPAMPoolScopeVNFService:
		entityName = vsName
	}
	vxlanIpamPool, err := s.IPAMPoolFindAllocator(poolName, entityName)
	if err != nil {
		return "", fmt.Errorf("Cannot find ipam pool %s: %s", poolName, err)
	}
	ipAddress, _, err := vxlanIpamPool.AllocateIPAddress()
	if err != nil {
		return "", fmt.Errorf("Cannot allocate address from ipamn pool %s", poolName)
	}
	return ipAddress, nil
}

func contructAllocatorName(ipamPool *controller.IPAMPool, entityName string) string {
	switch ipamPool.Scope {
	case controller.IPAMPoolScopeSystem:
		return fmt.Sprintf("/%s/%s", ipamPool.Scope, ipamPool.Name)
	case controller.IPAMPoolScopeNode:
		return fmt.Sprintf("/%s/%s/%s", ipamPool.Scope, ipamPool.Name, entityName)
	case controller.IPAMPoolScopeVNFService:
		return fmt.Sprintf("/%s/%s/%s", ipamPool.Scope, ipamPool.Name, entityName)
	}
	return ""
}

// IPAMPoolEntityCreate ensures a "scope" level pool allocator exists for this entity
func (s *Plugin) IPAMPoolEntityCreate(entityName string) {

	for _, ipamPool := range s.ramConfigCache.IPAMPools {
		ipamPoolAllocator, err := s.IPAMPoolFindAllocator(ipamPool.Name, entityName)
		if err != nil {
			ipamPoolAllocator = ipam.NewIPAMPoolAllocator(&ipamPool)
			allocatorName := contructAllocatorName(&ipamPool, entityName)
			s.ramConfigCache.IPAMPoolAllocators[allocatorName] = ipamPoolAllocator
		}
	}

}

// IPAMPoolEntityDelete removes a "scope" level pool allocator for this entity
func (s *Plugin) IPAMPoolEntityDelete(entityName string) {

	for _, ipamPool := range s.ramConfigCache.IPAMPools {
		ipamPoolAllocator, _ := s.IPAMPoolFindAllocator(ipamPool.Name, entityName)
		if ipamPoolAllocator != nil {
			allocatorName := contructAllocatorName(&ipamPool, entityName)
			delete(s.ramConfigCache.IPAMPoolAllocators, allocatorName)
		}
	}
	
}

