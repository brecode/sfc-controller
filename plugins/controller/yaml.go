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

// This config yaml file is loaded into a data structure and pulled in by the
// controller.

package controller

import (
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/ligato/sfc-controller/plugins/controller/model"
)

// YamlConfig is container struct for yaml config file
type YamlConfig struct {
	Version          int                         `json:"sfc_controller_config_version"`
	Description      string                      `json:"description"`
	Nodes            []controller.Node           `json:"nodes"`
	VNFServices      []controller.VNFService     `json:"vnf_services"`
	SysParms         controller.SystemParameters `json:"system_parameters"`
	VNFToNodeMap     []controller.VNFToNodeMap   `json:"vnf_to_node_map"`
	VNFServiceMeshes []controller.VNFServiceMesh `json:"vnf_service_meshes"`
	IPAMPools        []controller.IPAMPool       `json:"ipam_pools"`
}

// ReadYamlConfigFromFile parses the yaml into YamlConfig
func (s *Plugin) ReadYamlConfigFromFile(fpath string) (*YamlConfig, error) {

	b, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}

	yamlConfig := &YamlConfig{}

	if err := yaml.Unmarshal(b, yamlConfig); err != nil {
		return nil, err
	}

	return yamlConfig, nil
}

// ProcessYamlConfig processes each object and adds it to the system
func (s *Plugin) ProcessYamlConfig(y *YamlConfig) error {

	if y.Version != 2 {
		return fmt.Errorf("ProcessYamlConfig: incorrect yaml version, expecting 2, got: %d",
			y.Version)
	}

	log.Debugf("ProcessYamlConfig: system paramters: ", y.SysParms)
	if err := s.SysParmsCreate(&y.SysParms, false); err != nil {
		return err
	}

	log.Debugf("ProcessYamlConfig: ipam pools: ", y.IPAMPools)
	for _, ipamPool := range y.IPAMPools {
		if err := s.IPAMPoolCreate(&ipamPool, false); err != nil {
			return err
		}
	}

	for _, n := range y.Nodes {
		log.Debugf("ProcessYamlConfig: node: ", n)
		if err := s.NodeCreate(&n, false); err != nil {
			return err
		}
	}

	for _, v := range y.VNFServices {
		log.Debugf("ProcessYamlConfig: vnf-service: ", v)
		if err := s.VNFServiceCreate(&v, false); err != nil {
			return err
		}
	}

	log.Debugf("ProcessYamlConfig: vnf-to-node-map: ", y.VNFToNodeMap)
	if err := s.VNFToNodeMapCreate(y.VNFToNodeMap, false); err != nil {
		return err
	}

	log.Debugf("ProcessYamlConfig: vnf-service-meshes: ", y.VNFServiceMeshes)
	if err := s.VNFServiceMeshesCreate(y.VNFServiceMeshes, false); err != nil {
		return err
	}

	return nil
}
