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

// SfcControllerPrefix is the base for all entries
func SfcControllerPrefix() string {
	return "/sfc-controller/"
}
// SfcControllerConfigPrefix provides sfc controller prefix
func SfcControllerConfigPrefix() string {
	return SfcControllerPrefix() + "v2/config/"
}

// SfcControllerStatusPrefix provides sfc controller prefix
func SfcControllerStatusPrefix() string {
	return SfcControllerPrefix() + "v2/status/"
}

// SfcControllerContivKSRPrefix is the base for all contiv ksr entries
func SfcControllerContivKSRPrefix() string {
	return "/vnf-agent/contiv-ksr"
}

// SystemParametersKey provides sfc controller's system parameter key
func SystemParametersKey() string {
	return SfcControllerConfigPrefix() + "system-parameters"
}

// InterfaceStateKeyPrefix provides sfc controller's interface key prefix
func InterfaceStateKeyPrefix() string {
	return SfcControllerStatusPrefix() + "interface/"
}

// InterfacesStateHTTPPrefix provides sfc controller's interface retrieve HTTP prefix
func InterfacesStateHTTPPrefix() string {
	return SfcControllerStatusPrefix() + "interfaces"
}

// InterfaceStateKey provides sfc controller's interface key prefix
func InterfaceStateKey(vnf string, iface string) string {
	return InterfaceStateKeyPrefix() + vnf + "/" + iface
}

// NodeKeyPrefix provides sfc controller's node key prefix
func NodeKeyPrefix() string {
	return SfcControllerConfigPrefix() + "node/"
}

// NodesHTTPPrefix provides sfc controller's node retrieve HTTP prefix
func NodesHTTPPrefix() string {
	return SfcControllerConfigPrefix() + "nodes"
}

// NodeNameKey provides sfc controller's node name key prefix
func NodeNameKey(name string) string {
	return NodeKeyPrefix() + name
}

// VNFServiceKeyPrefix provides sfc controller's node key prefix
func VNFServiceKeyPrefix() string {
	return SfcControllerConfigPrefix() + "vnf-service/"
}

// VNFServicesHTTPPrefix provides sfc controller's node retrieve HTTP prefix
func VNFServicesHTTPPrefix() string {
	return SfcControllerConfigPrefix() + "vnf-services"
}

// VNFServiceNameKey provides sfc controller's node name key prefix
func VNFServiceNameKey(name string) string {
	return VNFServiceKeyPrefix() + name
}

// VNFToNodeKeyPrefix provides sfc controller's node key prefix
func VNFToNodeKeyPrefix() string {
	return SfcControllerConfigPrefix() + "vnf-to-node/"
}

// VNFToNodeMapHTTPPrefix provides sfc controller's vnf-node-map prefix
func VNFToNodeMapHTTPPrefix() string {
	return SfcControllerConfigPrefix() + "vnf-to-node-map"
}

// VNFToNodeKey provides sfc controller's vnf key prefix
func VNFToNodeKey(vnf string) string {
	return VNFToNodeKeyPrefix() + vnf
}

// VNFToNodeKeyStatusPrefix provides sfc controller's node key prefix
func VNFToNodeKeyStatusPrefix() string {
	return SfcControllerContivKSRPrefix() + SfcControllerStatusPrefix() + "vnf-to-node/"
}

// VNFToNodeMapStatusHTTPPrefix provides sfc controller's vnf-node-map prefix
func VNFToNodeMapStatusHTTPPrefix() string {
	return SfcControllerContivKSRPrefix() + SfcControllerStatusPrefix() + "vnf-to-node-map"
}

// VNFToNodeStatusKey provides sfc controller's vnf key prefix
func VNFToNodeStatusKey(vnf string) string {
	return VNFToNodeKeyStatusPrefix() + vnf
}

// VNFServiceKeyStatusPrefix provides sfc controller's VNFService key prefix
func VNFServiceKeyStatusPrefix() string {
	return SfcControllerStatusPrefix() + "vnf-service/"
}

// VNFServicesStatusHTTPPrefix provides sfc controller's v-service retrieve HTTP prefix
func VNFServicesStatusHTTPPrefix() string {
	return SfcControllerStatusPrefix() + "vnf-services"
}

// VNFServiceStatusNameKey provides sfc controller's node name key prefix
func VNFServiceStatusNameKey(name string) string {
	return VNFServiceKeyStatusPrefix() + name
}

// NodeKeyStatusPrefix provides sfc controller's node key prefix
func NodeKeyStatusPrefix() string {
	return SfcControllerStatusPrefix() + "node/"
}

// NodesStatusHTTPPrefix provides sfc controller's nodes retrieve HTTP prefix
func NodesStatusHTTPPrefix() string {
	return SfcControllerStatusPrefix() + "nodes"
}

// NodeStatusNameKey provides sfc controller's node name key prefix
func NodeStatusNameKey(name string) string {
	return NodeKeyStatusPrefix() + name
}

// VPPEntriesHTTPPrefix provides sfc controller's vpp-entries retrieve HTTP prefix
func VPPEntriesHTTPPrefix() string {
	return SfcControllerStatusPrefix() + "vpp-entries"
}

// VNFServiceMeshPrefix provides sfc controller's key prefix
func VNFServiceMeshPrefix() string {
	return SfcControllerConfigPrefix() + "vnf-service-mesh/"
}

// VNFServiceMeshesHTTPPrefix provides sfc controller's retrieve HTTP prefix
func VNFServiceMeshesHTTPPrefix() string {
	return SfcControllerConfigPrefix() + "vnf-service-meshes"
}

// VNFServiceMeshNameKey provides sfc controller's name key prefix
func VNFServiceMeshNameKey(name string) string {
	return VNFServiceMeshPrefix() + name
}

// IPAMPoolPrefix provides sfc controller's ipam pool key prefix
func IPAMPoolPrefix() string {
	return SfcControllerConfigPrefix() + "ipam-pool/"
}

// IPAMPoolsHTTPPrefix provides sfc controller's ipam pools retrieve HTTP prefix
func IPAMPoolsHTTPPrefix() string {
	return SfcControllerConfigPrefix() + "ipam-pools"
}

// IPAMPoolNameKey provides sfc controller's ipam pool name key prefix
func IPAMPoolNameKey(name string) string {
	return IPAMPoolPrefix() + name
}