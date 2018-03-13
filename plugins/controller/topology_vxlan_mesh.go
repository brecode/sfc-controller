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

// renderToplogySegmentLMPPInterNode renders this L2MP connection between nodes
func (s *Plugin) renderToplogyVxlanMesh(vs *controller.VNFService,
	conn *controller.Connection,
	connIndex uint32,
	vnfInterfaces []*controller.Interface,
	nodeOverlay *controller.NodeOverlay,
	v2n []controller.VNFToNodeMap,
	vnfTypes []string,
	nodeMap map[string]bool,
	l2bdIFs map[string][]*l2.BridgeDomains_BridgeDomain_Interfaces,
	vsState *controller.VNFServiceState) error {

	// The nodeMap contains the set of nodes involved in teh l2mp connection.  There
	// must be a vxlan mesh created between the nodes.  On each node, the vnf interfaces
	// will join the ldbd created for this connection, and the vxlan endpoint created
	// below is also associated with this bridge.

	// create the vxlan endpoints
	vniAllocator, exists := s.ramConfigCache.NodeOverlayVniAllocators[nodeOverlay.Name]
	if !exists {
		msg := fmt.Sprintf("vnf-service: %s, conn: %d, overlay: %s out of vni's",
			vs.Name,
			connIndex,
			nodeOverlay.Name)
		s.AppendStatusMsgToVnfService(msg, vsState)
		return fmt.Errorf(msg)
	}
	vni, err := vniAllocator.AllocateVni()
	if err != nil {
		msg := fmt.Sprintf("vnf-service: %s, conn: %d, overlay: %s out of vni's",
			vs.Name,
			connIndex,
			nodeOverlay.Name)
		s.AppendStatusMsgToVnfService(msg, vsState)
		return fmt.Errorf(msg)
	}


	for fromNode := range nodeMap {
		for toNode := range nodeMap {
			if fromNode == toNode {
				continue
			}

			ifName := fmt.Sprintf("IF_VXLAN_MESH_FROM_%s_TO_%s_VSRVC_%s_CONN_%d_VNI_%d",
				fromNode, toNode, vs.Name, connIndex, vni)

			vxlanIPFromAddress, err := s.NodeOverlayAllocateVxlanAddress(
				nodeOverlay.VxlanMeshParms.LoopbackIpamPoolName, fromNode)
			if err != nil {
				msg := fmt.Sprintf("vnf-service: %s, conn: %d, overlay: %s %s",
					vs.Name,
					connIndex,
					nodeOverlay.Name, err)
				s.AppendStatusMsgToVnfService(msg, vsState)
				return fmt.Errorf(msg)
			}
			vxlanIPToAddress, err := s.NodeOverlayAllocateVxlanAddress(
				nodeOverlay.VxlanMeshParms.LoopbackIpamPoolName, toNode)
			if err != nil {
				msg := fmt.Sprintf("vnf-service: %s, conn: %d, overlay: %s %s",
					vs.Name,
					connIndex,
					nodeOverlay.Name, err)
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
				nodeOverlay.VxlanMeshParms.OutgoingInterfaceLabel)

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
