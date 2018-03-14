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

	respChan := make(chan keyval.ProtoWatchResp, 0)

	// Register watcher and select the respChan channel as the destination
	// for the delivery of all the change events.
	watcher := s.Etcd.NewWatcher(controller.VNFToNodeKeyPrefix())
	
	//err := watcher.Watch(keyval.ToChanProto(respChan), make(chan string), controller.VNFToNodeKeyPrefix())
	err := watcher.Watch(keyval.ToChanProto(respChan), make(chan string), "")
	if err != nil {
		log.Errorf("RunVnfToNodeMappingWatcher: cannot watch: %s", err)
		os.Exit(1)
	}
	log.Debugf("RunVnfToNodeMappingWatcher: watching the key: %s", controller.VNFToNodeKeyPrefix())

	for {
		select {
		case resp := <-respChan:
			switch resp.GetChangeType() {
			case datasync.Put:
				//contact := &phonebook.Contact{}
				//prevContact := &phonebook.Contact{}
				log.Infof("Creating ", resp.GetKey())
				//resp.GetValue(contact)
				// exists, err := resp.GetPrevValue(prevContact)
				// if err != nil {
				// 	logrus.DefaultLogger().Errorf("err: %v\n", err)
				// }
				// printContact(contact)
				// if exists {
				// 	printPrevContact(prevContact)
				// } else {
				// 	fmt.Printf("Previous value does not exist\n")
				// }
			case datasync.Delete:
				log.Infof("Removing ", resp.GetKey())
				// prevContact := &phonebook.Contact{}
				// exists, err := resp.GetPrevValue(prevContact)
				// if err != nil {
				// 	logrus.DefaultLogger().Errorf("err: %v\n", err)
				// }
				// if exists {
				// 	printPrevContact(prevContact)
				// } else {
				// 	fmt.Printf("Previous value does not exist\n")
				// }
			}
		// case <-sigChan:
		// 	break watcherLoop
		}
	}
}
