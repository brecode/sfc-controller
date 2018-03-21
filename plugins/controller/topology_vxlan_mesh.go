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

	"github.com/ligato/sfc-controller/plugins/controller/model"
	"github.com/ligato/sfc-controller/plugins/controller/vppagentapi"
	"github.com/ligato/vpp-agent/plugins/defaultplugins/common/model/l2"
)

// renderToplogyL2MPVxlanMesh renders these L2MP tunnels between nodes
func (s *Plugin) renderToplogyL2MPVxlanMesh(vs *controller.VNFService,
	conn *controller.Connection,
	connIndex uint32,
	vnfInterfaces []*controller.Interface,
	vnfServiceMesh *controller.VNFServiceMesh,
	v2n []controller.VNFToNodeMap,
	vnfTypes []string,
	nodeMap map[string]bool,
	l2bdIFs map[string][]*l2.BridgeDomains_BridgeDomain_Interfaces,
	vsState *controller.VNFServiceState) error {

	// The nodeMap contains the set of nodes involved in the l2mp connection.  There
	// must be a vxlan mesh created between the nodes.  On each node, the vnf interfaces
	// will join the l2bd created for this connection, and the vxlan endpoint created
	// below is also associated with this bridge.

	// create the vxlan endpoints
	vniAllocator, exists := s.ramConfigCache.VNFServiceMeshVniAllocators[vnfServiceMesh.Name]
	if !exists {
		msg := fmt.Sprintf("vnf-service: %s, conn: %d, service mesh: %s out of vni's",
			vs.Name,
			connIndex,
			vnfServiceMesh.Name)
		s.AppendStatusMsgToVnfService(msg, vsState)
		return fmt.Errorf(msg)
	}
	vni, err := vniAllocator.AllocateVni()
	if err != nil {
		msg := fmt.Sprintf("vnf-service: %s, conn: %d, service mesh: %s out of vni's",
			vs.Name,
			connIndex,
			vnfServiceMesh.Name)
		s.AppendStatusMsgToVnfService(msg, vsState)
		return fmt.Errorf(msg)
	}

	// create a vxlan tunnel between each "from" node and "to" node
	for fromNode := range nodeMap {

		for toNode := range nodeMap {

			if fromNode == toNode {
				continue
			}

			ifName := fmt.Sprintf("IF_VXLAN_MESH_VSRVC_%s_CONN_%d_FROM_%s_TO_%s_VNI_%d",
				vs.Name, connIndex+1, fromNode, toNode, vni)

			vxlanIPFromAddress, err := s.VNFServiceMeshAllocateVxlanAddress(
				vnfServiceMesh.VxlanMeshParms.LoopbackIpamPoolName, fromNode)
			if err != nil {
				msg := fmt.Sprintf("vnf-service: %s, conn: %d, service mesh: %s %s",
					vs.Name,
					connIndex,
					vnfServiceMesh.Name, err)
				s.AppendStatusMsgToVnfService(msg, vsState)
				return fmt.Errorf(msg)
			}
			vxlanIPToAddress, err := s.VNFServiceMeshAllocateVxlanAddress(
				vnfServiceMesh.VxlanMeshParms.LoopbackIpamPoolName, toNode)
			if err != nil {
				msg := fmt.Sprintf("vnf-service: %s, conn: %d, service mesh: %s %s",
					vs.Name,
					connIndex,
					vnfServiceMesh.Name, err)
				s.AppendStatusMsgToVnfService(msg, vsState)
				return fmt.Errorf(msg)
			}

			vppKV := vppagentapi.ConstructVxlanInterface(
				fromNode,
				ifName,
				vni,
				vxlanIPFromAddress,
				vxlanIPToAddress)
			vsState.RenderedVppAgentEntries =
				s.ConfigTransactionAddVppEntry(vsState.RenderedVppAgentEntries, vppKV)

			l2bdIF := &l2.BridgeDomains_BridgeDomain_Interfaces{
				Name: ifName,
				BridgedVirtualInterface: false,
				SplitHorizonGroup:       1,
			}
			l2bdIFs[fromNode] = append(l2bdIFs[fromNode], l2bdIF)

			renderedEntries := s.NodeRenderVxlanStaticRoutes(fromNode, toNode,
				vxlanIPFromAddress, vxlanIPToAddress,
				vnfServiceMesh.VxlanMeshParms.OutgoingInterfaceLabel)

			vsState.RenderedVppAgentEntries = append(vsState.RenderedVppAgentEntries,
				renderedEntries...)
		}
	}

	// create the perNode lsbd's and add the vnf interfaces
	for nodeName := range nodeMap {
		if err := s.renderL2BD(vs, conn, connIndex, nodeName, l2bdIFs[nodeName], vsState); err != nil {
			return err
		}
	}

	return nil
}

// renderToplogyL2PPVxlanMesh renders this L2PP tunnel between nodes
func (s *Plugin) renderToplogyL2PPVxlanMesh(vs *controller.VNFService,
	conn *controller.Connection,
	connIndex uint32,
	vnfInterfaces []*controller.Interface,
	vnfServiceMesh *controller.VNFServiceMesh,
	v2n [2]controller.VNFToNodeMap,
	xconn [2][2]string,
	vsState *controller.VNFServiceState) error {

	// The nodeMap contains the set of nodes involved in the l2mp connection.  There
	// must be a vxlan mesh created between the nodes.  On each node, the vnf interfaces
	// will join the l2bd created for this connection, and the vxlan endpoint created
	// below is also associated with this bridge.

	// create the vxlan endpoints
	vniAllocator, exists := s.ramConfigCache.VNFServiceMeshVniAllocators[vnfServiceMesh.Name]
	if !exists {
		msg := fmt.Sprintf("vnf-service: %s, conn: %d, %s/%s to %s/%s service mesh: %s out of vni's",
			vs.Name,
			connIndex,
			conn.Interfaces[0].Vnf, conn.Interfaces[0].Interface,
			conn.Interfaces[1].Vnf, conn.Interfaces[1].Interface,
			vnfServiceMesh.Name)
		s.AppendStatusMsgToVnfService(msg, vsState)
		return fmt.Errorf(msg)
	}
	vni, err := vniAllocator.AllocateVni()
	if err != nil {
		msg := fmt.Sprintf("vnf-service: %s, conn: %d, %s/%s to %s/%s service mesh: %s out of vni's",
			vs.Name,
			connIndex,
			conn.Interfaces[0].Vnf, conn.Interfaces[0].Interface,
			conn.Interfaces[1].Vnf, conn.Interfaces[1].Interface,
			vnfServiceMesh.Name)
		s.AppendStatusMsgToVnfService(msg, vsState)
		return fmt.Errorf(msg)
	}

	for i := 0; i < 2; i++ {

		from := i
		to := ^i&1

		ifName := fmt.Sprintf("IF_VXLAN_L2PP_VSRVC_%s_CONN_%d_FROM_%s_%s_%s_TO_%s_%s_%s_VNI_%d",
			vs.Name, connIndex+1,
			v2n[from].Node, conn.Interfaces[from].Vnf, conn.Interfaces[from].Interface,
			v2n[to].Node, conn.Interfaces[to].Vnf, conn.Interfaces[to].Interface,
			vni)

		xconn[1][i] = ifName

		vxlanIPFromAddress, err := s.VNFServiceMeshAllocateVxlanAddress(
			vnfServiceMesh.VxlanMeshParms.LoopbackIpamPoolName, v2n[i].Node)
		if err != nil {
			msg := fmt.Sprintf("vnf-service: %s, conn: %d, %s/%s to %s/%s service mesh: %s, %s",
				vs.Name,
				connIndex,
				conn.Interfaces[0].Vnf, conn.Interfaces[0].Interface,
				conn.Interfaces[1].Vnf, conn.Interfaces[1].Interface,
				vnfServiceMesh.Name, err)
			s.AppendStatusMsgToVnfService(msg, vsState)
			return fmt.Errorf(msg)
		}
		vxlanIPToAddress, err := s.VNFServiceMeshAllocateVxlanAddress(
			vnfServiceMesh.VxlanMeshParms.LoopbackIpamPoolName, v2n[^i&1].Node)
		if err != nil {
			msg := fmt.Sprintf("vnf-service: %s, conn: %d, %s/%s to %s/%s service mesh: %s %s",
				vs.Name,
				connIndex,
				conn.Interfaces[0].Vnf, conn.Interfaces[0].Interface,
				conn.Interfaces[1].Vnf, conn.Interfaces[1].Interface,
				vnfServiceMesh.Name, err)
			s.AppendStatusMsgToVnfService(msg, vsState)
			return fmt.Errorf(msg)
		}

		vppKV := vppagentapi.ConstructVxlanInterface(
			v2n[i].Node,
			ifName,
			vni,
			vxlanIPFromAddress,
			vxlanIPToAddress)
		vsState.RenderedVppAgentEntries =
			s.ConfigTransactionAddVppEntry(vsState.RenderedVppAgentEntries, vppKV)

		renderedEntries := s.NodeRenderVxlanStaticRoutes(v2n[i].Node, v2n[^i&1].Node,
			vxlanIPFromAddress, vxlanIPToAddress,
			vnfServiceMesh.VxlanMeshParms.OutgoingInterfaceLabel)

		vsState.RenderedVppAgentEntries = append(vsState.RenderedVppAgentEntries,
			renderedEntries...)
	}

	// create xconns between vswitch side of the container interfaces and the vxlan ifs
	for i := 0; i < 2; i++ {
		vppKVs := vppagentapi.ConstructXConnect(v2n[i].Node, xconn[0][i], xconn[1][i])
		vsState.RenderedVppAgentEntries =
			s.ConfigTransactionAddVppEntries(vsState.RenderedVppAgentEntries, vppKVs)
	}

	return nil
}
