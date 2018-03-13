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
	"github.com/ligato/sfc-controller/plugins/controller/model"
)

func (s *Plugin) findVnfInAnyVNFService(vnfName string) bool {
	for _, vs := range s.ramConfigCache.VNFServices {
		for _, vnf := range vs.Vnfs {
			if vnf.Name == vnfName {
				return true
			}
		}
	}
	return false
}

// VNFToNodeMapValidate sees if the entites involved exist yet
func (s *Plugin) VNFToNodeMapValidate(v2n *controller.VNFToNodeMap) error {

	if _, exists := s.ramConfigCache.Nodes[v2n.Node]; !exists {
		log.Warnf("VNFToNodeMapValidate: v2n-map node not defined yet", v2n)
	}

	if exists := s.findVnfInAnyVNFService(v2n.Vnf); !exists {
		log.Warnf("VNFToNodeMapValidate: v2n-map vnf not found in a service yet", v2n)
	}

	return nil
}

// VNFToNodeMapCreate add to ram cache and run topology side effects
func (s *Plugin) VNFToNodeMapCreate(v2nMap []controller.VNFToNodeMap, runSideEffects bool) error {

	// when a vnf is discovered and associated with a node, then we
	// can traverse the vnf-services to see if a service is ready to
	// be rendered

	for _, v2n := range v2nMap {
		if err := s.VNFToNodeMapValidate(&v2n); err != nil {
			return err
		}

		s.ramConfigCache.VNFToNodeMap[v2n.Vnf] = v2n

		if err := s.VNFToNodeWriteToDatastore(&v2n); err != nil {
			return err
		}
	}

	// process all vnfs and re-render vpp agent configs in case a vnf moved from
	// one host to another
	if runSideEffects {
		if err := s.VNFServicesRender(); err != nil {
			return err
		}
	}

	return nil
}
