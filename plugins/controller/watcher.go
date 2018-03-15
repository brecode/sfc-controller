// Copyright (c) 2018 Cisco and/or its affiliates.
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

//go:generate protoc --proto_path=model --gogo_out=model model/controller.proto

// The core plugin which drives the SFC Controller.  The core initializes the
// CNP dirver plugin based on command line args.  The database is initialized,
// and a resync is preformed based on what was already in the database.

package controller

import (
	"os"
	"time"
	"github.com/ligato/cn-infra/datasync"
	"github.com/ligato/cn-infra/db/keyval"
	"github.com/ligato/sfc-controller/plugins/controller/model"
)

// StartWatchers initializes etcd tree specific watchers
func (s *Plugin) StartWatchers() {
	go s.RunVnfToNodeMappingWatcher()
}

// RunVnfToNodeMappingWatcher enables etcd updates to be monitored
func (s *Plugin) RunVnfToNodeMappingWatcher() {

	log.Info("RunVnfToNodeMappingWatcher: enter ...")
	defer log.Info("RunVnfToNodeMappingWatcher: exit ...")

    go func() {
		// back up timer ... paranoid about missing events ...
		// check every minute just in case
		ticker := time.NewTicker(1 * time.Minute)
        for _ = range ticker.C {
			tempV2NStateMap := make(map[string]controller.VNFToNodeMap)
			s.LoadVNFToNodeMapStateFromDatastore(tempV2NStateMap)
			renderingRequired := false
			for _, tempV2NMap := range tempV2NStateMap {
				v2nMap, exists := s.ramConfigCache.VNFToNodeStateMap[tempV2NMap.Vnf]
				//log.Debugf("RunVnfToNodeMappingWatcher: timer v2n: %v", v2nMap)
				if !exists || v2nMap.Node !=  tempV2NMap.Node {
					renderingRequired = true
					s.VNFToNodeStateCreate(&tempV2NMap, false)
				}
			}
			if renderingRequired {
				s.RenderConfig()
			}
			tempV2NStateMap = nil
        }
    }()

	respChan := make(chan keyval.ProtoWatchResp, 0)
	watcher := s.Etcd.NewWatcher(controller.VNFToNodeKeyStatusPrefix())
	err := watcher.Watch(keyval.ToChanProto(respChan), make(chan string), "")
	if err != nil {
		log.Errorf("RunVnfToNodeMappingWatcher: cannot watch: %s", err)
		os.Exit(1)
	}
	log.Debugf("RunVnfToNodeMappingWatcher: watching the key: %s", controller.VNFToNodeKeyStatusPrefix())

	for {
		select {
		case resp := <-respChan:
			switch resp.GetChangeType() {
			case datasync.Put:
				v2n := &controller.VNFToNodeMap{}
				if err :=resp.GetValue(v2n); err == nil {
					log.Infof("RunVnfToNodeMappingWatcher: key: %s, value:%v", resp.GetKey(), v2n)
					s.VNFToNodeStateCreate(v2n, true)
				}

			case datasync.Delete:
				log.Infof("RunVnfToNodeMappingWatcher: deleting key: %s ", resp.GetKey())
				s.VNFToNodeStateDelete(resp.GetKey(), true)
			}
		}
	}
}
