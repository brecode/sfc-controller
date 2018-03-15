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
	"github.com/golang/protobuf/proto"
	"github.com/ligato/cn-infra/datasync"
	"github.com/ligato/sfc-controller/plugins/controller/model"
)

// SysParmsWriteToDatastore writes the specified entity in the sfc db in etcd
func (s *Plugin) SysParmsWriteToDatastore(sp *controller.SystemParameters) error {
	key := controller.SystemParametersKey()
	return s.writeToDatastore(key, sp)
}

// NodeWriteToDatastore writes the specified entity in the sfc db in etcd
func (s *Plugin) NodeWriteToDatastore(n *controller.Node) error {
	key := controller.NodeNameKey(n.Name)
	return s.writeToDatastore(key, n)
}

// InterfaceStateWriteToDatastore writes the specified entity in the sfc db in etcd
func (s *Plugin) InterfaceStateWriteToDatastore(ifState *controller.InterfaceState) error {
	key := controller.InterfaceStateKey(ifState.Vnf, ifState.Interface)
	return s.writeToDatastore(key, ifState)
}

// NodeStateWriteToDatastore writes the specified entity in the sfc db in etcd
func (s *Plugin) NodeStateWriteToDatastore(n *controller.NodeState) error {
	key := controller.NodeStatusNameKey(n.Name)
	return s.writeToDatastore(key, n)
}

// VNFServiceStatusWriteToDatastore writes the specified entity in the sfc db in etcd
func (s *Plugin) VNFServiceStatusWriteToDatastore(vs *controller.VNFServiceState) error {
	key := controller.VNFServiceStatusNameKey(vs.Name)
	return s.writeToDatastore(key, vs)
}

// VNFServiceWriteToDatastore writes the specified entity in the sfc db in etcd
func (s *Plugin) VNFServiceWriteToDatastore(vs *controller.VNFService) error {
	key := controller.VNFServiceNameKey(vs.Name)
	return s.writeToDatastore(key, vs)
}

// VNFServiceMeshWriteToDatastore writes the specified entity in the sfc db in etcd
func (s *Plugin) VNFServiceMeshWriteToDatastore(no *controller.VNFServiceMesh) error {
	key := controller.VNFServiceMeshNameKey(no.Name)
	return s.writeToDatastore(key, no)
}

// IPAMPoolWriteToDatastore writes the specified entity in the sfc db in etcd
func (s *Plugin) IPAMPoolWriteToDatastore(ipamPool *controller.IPAMPool) error {
	key := controller.IPAMPoolNameKey(ipamPool.Name)
	return s.writeToDatastore(key, ipamPool)
}

// VNFToNodeWriteToDatastore writes the specified entity in the sfc db in etcd
func (s *Plugin) VNFToNodeWriteToDatastore(v2n *controller.VNFToNodeMap) error {
	key := controller.VNFToNodeKey(v2n.Vnf)
	return s.writeToDatastore(key, v2n)
}

// writeToDatastore writes the specified entity in the sfc db in etcd
func (s *Plugin) writeToDatastore(key string, data proto.Message) error {

	log.Debugf("writeToDatastore: key: '%s'", key)

	err := s.db.Put(key, data)
	if err != nil {
		log.Error("DatastoreExternalEntityCreate: write error: ", err)
		return err
	}
	return nil
}

// LoadSfcConfigIntoRAMCache loads the etc objects into the ram tables
func (s *Plugin) LoadSfcConfigIntoRAMCache() error {

	log.Debugf("LoadSfcConfigIntoRAMCache: ...")

	if err := s.LoadSysParmsFromDatastore(&s.ramConfigCache.SysParms); err != nil {
		return err
	}
	if err := s.LoadAllNodesFromDatastore(s.ramConfigCache.Nodes); err != nil {
		return err
	}
	if err := s.LoadAllInterfaceStatesFromDatastore(s.ramConfigCache.InterfaceStates); err != nil {
		return err
	}
	if err := s.LoadAllVNFServicesFromDatastore(s.ramConfigCache.VNFServices); err != nil {
		return err
	}
	if err := s.LoadAllVNFServiceMeshesFromDatastore(s.ramConfigCache.VNFServiceMeshes); err != nil {
		return err
	}
	if err := s.LoadAllIPAMPoolsFromDatastore(s.ramConfigCache.IPAMPools); err != nil {
		return err
	}
	if err := s.LoadVNFToNodeMapFromDatastore(s.ramConfigCache.VNFToNodeMap); err != nil {
		return err
	}
	if err := s.LoadAllNodeStatesFromDatastore(s.ramConfigCache.NodeState); err != nil {
		return err
	}
	if err := s.LoadAllVNFServiceStatesFromDatastore(s.ramConfigCache.VNFServicesState); err != nil {
		return err
	}
	return nil
}

// LoadAllNodesFromDatastore iterates over the etcd set
func (s *Plugin) LoadAllNodesFromDatastore(nodes map[string]controller.Node) error {
	log.Debugf("LoadAllNodesFromDatastore: ...")
	n := &controller.Node{}
	return s.readIterate(controller.NodeKeyPrefix(),
		func() proto.Message { return n },
		func(data proto.Message) {
			nodes[n.Name] = *n
			log.Debugf("LoadAllNodesFromDatastore: n=%v", n)
		})
}

// LoadAllInterfaceStatesFromDatastore iterates over the etcd set
func (s *Plugin) LoadAllInterfaceStatesFromDatastore(interfaceStates map[string]controller.InterfaceState) error {
	log.Debugf("LoadAllInterfaceStatesFromDatastore: ...")
	ifState := &controller.InterfaceState{}
	return s.readIterate(controller.InterfaceStateKeyPrefix(),
		func() proto.Message { return ifState },
		func(data proto.Message) {
			interfaceStates[ifState.Vnf + "/" + ifState.Interface] = *ifState
			log.Debugf("LoadAllInterfaceStatesFromDatastore: ifState=%v", ifState)
		})
}


// LoadSysParmsFromDatastore reads the sysparms from the db
func (s *Plugin) LoadSysParmsFromDatastore(sp *controller.SystemParameters) error {
	log.Debugf("LoadSysParmsFromDatastore: ...")
	key := controller.SystemParametersKey()
	found, _, err := s.db.GetValue(key, sp)
	if found && err == nil {
		log.Debugf("LoadSysParmsFromDatastore: sp=%v", sp)
	}
	return err
}


// LoadAllVNFServicesFromDatastore iterates over the etcd set
func (s *Plugin) LoadAllVNFServicesFromDatastore(vservices map[string]controller.VNFService) error {
	log.Debugf("LoadAllVNFServicesFromDatastore: ...")
	v := &controller.VNFService{}
	return s.readIterate(controller.VNFServiceKeyPrefix(),
		func() proto.Message { return v },
		func(data proto.Message) {
			vservices[v.Name] = *v
			log.Debugf("LoadAllVNFServicesFromDatastore: v=%v", v)
		})
}

// LoadVNFToNodeMapFromDatastore iterates over the etcd set
func (s *Plugin) LoadVNFToNodeMapFromDatastore(v2nMap map[string]controller.VNFToNodeMap) error {
	log.Debugf("LoadVNFToNodeMapFromDatastore: ...")
	v2n := &controller.VNFToNodeMap{}
	return s.readIterate(controller.VNFToNodeKeyPrefix(),
		func() proto.Message { return v2n },
		func(data proto.Message) {
			v2nMap[v2n.Vnf] = *v2n
			log.Debugf("LoadVNFToNodeMapFromDatastore: v=%v", v2n)
		})
}

// LoadVNFToNodeMapStateFromDatastore iterates over the etcd set
func (s *Plugin) LoadVNFToNodeMapStateFromDatastore(v2nMap map[string]controller.VNFToNodeMap) error {
	//log.Debugf("LoadVNFToNodeMapStateFromDatastore: ...")
	v2n := &controller.VNFToNodeMap{}
	return s.readIterate(controller.VNFToNodeKeyStatusPrefix(),
		func() proto.Message { return v2n },
		func(data proto.Message) {
			v2nMap[v2n.Vnf] = *v2n
			//log.Debugf("LoadVNFToNodeMapStateFromDatastore: v=%v", v2n)
		})
}
// LoadAllVNFServiceMeshesFromDatastore iterates over the etcd set
func (s *Plugin) LoadAllVNFServiceMeshesFromDatastore(vnfServiceMeshes map[string]controller.VNFServiceMesh) error {
	log.Debugf("LoadAllVNFServiceMeshesFromDatastore: ...")
	vsm := &controller.VNFServiceMesh{}
	return s.readIterate(controller.VNFServiceMeshPrefix(),
		func() proto.Message { return vsm },
		func(data proto.Message) {
			vnfServiceMeshes[vsm.Name] = *vsm
			log.Debugf("LoadAllVNFServiceMeshesFromDatastore: vsm=%v", vsm)
		})
}

// LoadAllIPAMPoolsFromDatastore iterates over the etcd set
func (s *Plugin) LoadAllIPAMPoolsFromDatastore(ipamPools map[string]controller.IPAMPool) error {
	log.Debugf("LoadAllIPAMPoolsFromDatastore: ...")
	ipamPool := &controller.IPAMPool{}
	return s.readIterate(controller.IPAMPoolPrefix(),
		func() proto.Message { return ipamPool },
		func(data proto.Message) {
			ipamPools[ipamPool.Name] = *ipamPool
			log.Debugf("LoadAllIPAMPoolsFromDatastore: ipamPool=%v", ipamPool)
		})
}

// LoadAllNodeStatesFromDatastore iterates over the etcd set
func (s *Plugin) LoadAllNodeStatesFromDatastore(nodeStates map[string]*controller.NodeState) error {
	log.Debugf("LoadAllNodeStatesFromDatastore: ...")
	var n *controller.NodeState
	return s.readIterate(controller.NodeKeyStatusPrefix(),
		func() proto.Message {
			n = &controller.NodeState{}
			return n
		},
		func(data proto.Message) {
			nodeStates[n.Name] = n
			log.Debugf("LoadAllNodeStatesFromDatastore: ns=%v", n)
		})
}

// LoadAllVNFServiceStatesFromDatastore iterates over the etcd set
func (s *Plugin) LoadAllVNFServiceStatesFromDatastore(vsStates map[string]*controller.VNFServiceState) error {
	log.Debugf("LoadAllVNFServiceStatesFromDatastore: ...")
	var v *controller.VNFServiceState
	return s.readIterate(controller.VNFServiceKeyStatusPrefix(),
		func() proto.Message {
			v = &controller.VNFServiceState{}
			return v
		},
		func(data proto.Message) {
			vsStates[v.Name] = v
			log.Debugf("LoadAllVNFServiceStatesFromDatastore: vs=%v", v)
		})
}

func (s *Plugin) readIterate(
	keyPrefix string,
	getDataBuffer func() proto.Message,
	actionFunc func(data proto.Message)) error {

	kvi, err := s.db.ListValues(keyPrefix)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	for {
		kv, allReceived := kvi.GetNext()
		if allReceived {
			return nil
		}
		data := getDataBuffer()
		err := kv.GetValue(data)
		if err != nil {
			log.Fatal(err)
			return nil
		}
		actionFunc(data)
	}
}

// DeleteFromDatastore removes the specified entry fron etcd
func (s *Plugin) DeleteFromDatastore(key string) {

	log.Debugf("DeleteFromDatastore: key: '%s'", key)
	s.db.Delete(key)
}

// CleanSfcDatastore removes all entries from /sfc-controller and below
func (s *Plugin) CleanSfcDatastore() {
	s.db.Delete(controller.SfcControllerPrefix(), datasync.WithPrefix())
	log.Debugf("CleanSfcDatastore: clearing etc tree")
}
