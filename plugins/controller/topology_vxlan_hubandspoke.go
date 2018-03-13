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
func (s *Plugin) renderToplogyVxlanHubAndSpoke(vs *controller.VNFService,
	conn *controller.Connection,
	connIndex uint32,
	vnfInterfaces []*controller.Interface,
	nodeOverlay *controller.NodeOverlay,
	v2n []controller.VNFToNodeMap,
	vnfTypes []string,
	spokeNodeMap map[string]bool,
	l2bdIFs map[string][]*l2.BridgeDomains_BridgeDomain_Interfaces,
	vsState *controller.VNFServiceState) error {

	// The spokeNodeMap contains the set of nodes involved in the l2mp connection.  The node
	// in the node overlay is the hub and this set of nodes in the nodeMap are the spokes.
	// Need to create an l2bd on the hub node and add each of the vxlan tunnels to it, also
	// need to create an l2bd on each of the spoke nodes and add the vxlan spoke to it.
	// Note that the l2bdIFs map passed into this function in the input parameters already
	// has the per spoke vnf interfaces in it.

	hubNodeName := nodeOverlay.VxlanHubAndSpokeParms.HubNodeName
	if _, exists := s.ramConfigCache.Nodes[hubNodeName]; !exists {
		msg := fmt.Sprintf("vnf-service: %s, conn: %d, overlay: %s, hub_node: %s not found",
			vs.Name,
			connIndex,
			nodeOverlay.Name,
			hubNodeName)
		s.AppendStatusMsgToVnfService(msg, vsState)
		return fmt.Errorf(msg)
	}

	vni := nodeOverlay.VxlanHubAndSpokeParms.Vni

	// for each spoke, create a hub-spoke and spoke-hub vxlan i/f
	for spokeNodeName := range spokeNodeMap {

		if hubNodeName == spokeNodeName {
			msg := fmt.Sprintf("vnf-service: %s, conn: %d, overlay: %s hub node same as spoke %s",
				vs.Name,
				connIndex,
				nodeOverlay.Name, hubNodeName)
			s.AppendStatusMsgToVnfService(msg, vsState)
			return fmt.Errorf(msg)
		}

		// create both ends of the tunnel, hub end and spoke end
		hubAndSpokeNodePair := []string{hubNodeName, spokeNodeName}
		for i := range hubAndSpokeNodePair {

			fromNode := hubAndSpokeNodePair[i]
			toNode := hubAndSpokeNodePair[^i&1]

			var ifName string
			if i == 0 {
				ifName = fmt.Sprintf("IF_VXLAN_FROM_HUB_%s_TO_SPOKE_%s_VSRVC_%s_CONN_%d_VNI_%d",
					fromNode, toNode, vs.Name, connIndex, vni)
			} else {
				ifName = fmt.Sprintf("IF_VXLAN_FROM_SPOKE_%s_TO_HUB_%s_VSRVC_%s_CONN_%d_VNI_%d",
					fromNode, toNode, vs.Name, connIndex, vni)
			}

			vxlanIPFromAddress, err := s.NodeOverlayAllocateVxlanAddress(
				nodeOverlay.VxlanHubAndSpokeParms.LoopbackIpamPoolName, fromNode)
			if err != nil {
				msg := fmt.Sprintf("vnf-service: %s, conn: %d, overlay: %s %s",
					vs.Name,
					connIndex,
					nodeOverlay.Name, err)
				s.AppendStatusMsgToVnfService(msg, vsState)
				return fmt.Errorf(msg)
			}
			vxlanIPToAddress, err := s.NodeOverlayAllocateVxlanAddress(
				nodeOverlay.VxlanHubAndSpokeParms.LoopbackIpamPoolName, toNode)
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

			// internode VNFs reach each other via the hub node, need split-horizon 0
			l2bdIF := &l2.BridgeDomains_BridgeDomain_Interfaces{
				Name: ifName,
				BridgedVirtualInterface: false,
				SplitHorizonGroup:       0,
			}
			l2bdIFs[fromNode] = append(l2bdIFs[fromNode], l2bdIF)

			renderedEntries := s.NodeRenderVxlanStaticRoutes(fromNode, toNode,
				vxlanIPFromAddress, vxlanIPToAddress,
				nodeOverlay.VxlanHubAndSpokeParms.OutgoingInterfaceLabel)

			vsState.RenderedVppAgentEntries = append(vsState.RenderedVppAgentEntries,
				renderedEntries...)
		}
	}

	// create the spoke node l2bd's and add the vnf interfaces and vxlan if's from abve
	for nodeName := range spokeNodeMap {
		if err := s.renderL2BD(vs, conn, connIndex, nodeName, l2bdIFs[nodeName], vsState); err != nil {
			return err
		}
	}
	// create the hub l2bd and add the vxaln if's from above
	if err := s.renderL2BD(vs, conn, connIndex, hubNodeName, l2bdIFs[hubNodeName], vsState); err != nil {
		return err
	}

	return nil
}
