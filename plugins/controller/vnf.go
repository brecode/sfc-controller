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
	"github.com/ligato/sfc-controller/plugins/controller/model"
)

// ValidateVnfs validates all the fields
func (s *Plugin) ValidateVnfs(vsName string,
	vnf *controller.VNF) error {

	switch vnf.VnfType {
	case controller.VNFTypeExternal:
	case controller.VNFTypeVPPContainer:
	case controller.VNFTypeNonVPPContainer:
	default:
		return fmt.Errorf("vnfServiceValidateVnfs: vnf-service: %s, vnf: %s invalid vnf-type %s",
			vsName, vnf.Name, vnf.VnfType)
	}
	
	if len(vnf.Interfaces) == 0 {
		return fmt.Errorf("vnf-service/vnf: %s/%s no interfaces defined", vsName, vnf.Name)
	}

	for _, iFace := range vnf.Interfaces {
		switch iFace.IfType {
		case controller.IfTypeMemif:
		case controller.IfTypeEthernet:
		case controller.IfTypeVeth:
		case controller.IfTypeTap:
		default:
			return fmt.Errorf("vnf-service/vnf: %s/%s has invalid if type '%s'",
				vsName, iFace.Name, iFace.IfType)
		}
		for _, ipAddress := range iFace.IpAddresses {
			ip, network, err := net.ParseCIDR(ipAddress)
			if err != nil {
				return fmt.Errorf("vnf-service/if: %s/%s '%s', expected format i.p.v.4/xx, or ip::v6/xx",
					vsName, iFace.Name, err)
			}
			log.Debugf("ValidateVnfs: ip: %s, network: %s", ip, network)
		}
		if iFace.MemifParms != nil {
			if iFace.MemifParms.Mode != "" {
				switch iFace.MemifParms.Mode {
				case controller.IfMemifModeEhernet:
				case controller.IfMemifModeIP:
				case controller.IfMemifModePuntInject:
				default:
					return fmt.Errorf("vnf-service/if: %s/%s, unsupported memif mode=%s",
						vsName, iFace.Name, iFace.MemifParms.Mode)
				}
			}
			if iFace.MemifParms.InterVnfConn != "" {
				switch iFace.MemifParms.InterVnfConn {
				case controller.IfMemifInterVnfConnTypeDirect:
				case controller.IfMemifInterVnfConnTypeVswitch:
				default:
					return fmt.Errorf("vnf-service/if: %s/%s, unsupported memif inter-vnf connection type=%s",
						vsName, iFace.Name, iFace.MemifParms.InterVnfConn)
				}
			}
		}
	}

	return nil
}