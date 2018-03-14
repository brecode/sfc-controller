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

// The HTTP REST interface and implementation.  The model for the controller in
// ligato/sfc-controller/controller/model drives the REST interface.  The
// model is described in the protobuf file.  Each of the entites like hosts,
// external routers, and SFC's can be configrued (CRUDed) via REST calls.

package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ligato/sfc-controller/plugins/controller/model"
	"github.com/ligato/sfc-controller/plugins/controller/vppagentapi"
	"github.com/unrolled/render"
)

const (
	entityName = "entityName"
)

var sfcplg *Plugin

// InitHTTPHandlers : register the handler funcs for GET and POST, TODO PUT/DELETE
func (s *Plugin) InitHTTPHandlers() {

	sfcplg = s

	log.Infof("InitHTTPHandlers: registering ...")

	s.HTTPmux.RegisterHTTPHandler(controller.SystemParametersKey(),
		systemParametersHandler, "GET", "POST")

	log.Infof("InitHTTPHandlers: registering GET/POST %s", controller.NodeKeyPrefix())
	url := fmt.Sprintf(controller.NodeKeyPrefix()+"{%s}", entityName)
	s.HTTPmux.RegisterHTTPHandler(url, nodeHandler, "GET", "POST", "DELETE")

	log.Infof("InitHTTPHandlers: registering GET/POST %s", controller.VNFToNodeMapHTTPPrefix())
	s.HTTPmux.RegisterHTTPHandler(controller.VNFToNodeMapHTTPPrefix(),
		vnfToNodeMapHandler, "GET", "POST")

	log.Infof("InitHTTPHandlers: registering GET %s", controller.VNFServicesStatusHTTPPrefix())
	s.HTTPmux.RegisterHTTPHandler(controller.VNFServicesStatusHTTPPrefix(),
		VNFServicesStatusHandler, "GET")

	log.Infof("InitHTTPHandlers: registering GET %s", controller.VNFServicesHTTPPrefix())
	s.HTTPmux.RegisterHTTPHandler(controller.VNFServicesHTTPPrefix(), VNFServicesHandler, "GET")

	log.Infof("InitHTTPHandlers: registering GET %s", controller.NodesStatusHTTPPrefix())
	s.HTTPmux.RegisterHTTPHandler(controller.NodesStatusHTTPPrefix(), NodesStatusHandler, "GET")

	log.Infof("InitHTTPHandlers: registering GET %s", controller.NodesHTTPPrefix())
	s.HTTPmux.RegisterHTTPHandler(controller.NodesHTTPPrefix(), NodesHandler, "GET")

	log.Infof("InitHTTPHandlers: registering GET %s", controller.InterfacesStateHTTPPrefix())
	s.HTTPmux.RegisterHTTPHandler(controller.InterfacesStateHTTPPrefix(), InterfacesStateHandler, "GET")

	log.Infof("InitHTTPHandlers: registering GET %s", controller.VNFServiceMeshesHTTPPrefix())
	s.HTTPmux.RegisterHTTPHandler(controller.VNFServiceMeshesHTTPPrefix(), VNFServiceMeshesHandler, "GET")

	log.Infof("InitHTTPHandlers: registering GET %s", controller.IPAMPoolsHTTPPrefix())
	s.HTTPmux.RegisterHTTPHandler(controller.IPAMPoolsHTTPPrefix(), IPAMPoolsHandler, "GET")

	log.Infof("InitHTTPHandlers: registering GET %s", controller.VPPEntriesHTTPPrefix())
	s.HTTPmux.RegisterHTTPHandler(controller.VPPEntriesHTTPPrefix(), VPPEntriesHandler, "GET")

}

// Example curl invocations: for obtaining ALL external_entities
//   - GET:  curl http://localhost:9191/sfc-controller/v2/config/vnf-to-node-map
//   - POST: curl -X POST -d '[{"vnf":"vnf1", "node":"vswitch1"}]' http://localhost:9191/sfc_controller/v2/config/vnf-to-node-map
func vnfToNodeMapHandler(formatter *render.Render) http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		log.Debugf("VNF To Node Map HTTP handler: Method %s, URL: %s", req.Method, req.URL)

		switch req.Method {
		case "GET":
			var array = make([]controller.VNFToNodeMap, 0)
			for _, v2n := range sfcplg.ramConfigCache.VNFToNodeMap {
				array = append(array, v2n)
			}
			formatter.JSON(w, http.StatusOK, array)

		case "POST":
			processVnfToNodeMapPost(formatter, w, req)
		}
	}
}

func processVnfToNodeMapPost(formatter *render.Render, w http.ResponseWriter, req *http.Request) {

	sfcplg.ConfigTransactionStart()
	defer sfcplg.ConfigTransactionEnd()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Debugf("Can't read body, error '%s'", err)
		formatter.JSON(w, http.StatusBadRequest, struct{ Error string }{err.Error()})
		return
	}
	var v2nArray []controller.VNFToNodeMap
	err = json.Unmarshal(body, &v2nArray)
	if err != nil {
		log.Debugf("Can't parse body, error '%s'", err)
		formatter.JSON(w, http.StatusBadRequest, struct{ Error string }{err.Error()})
		return
	}
	log.Debugf("vnfToNodeMapHandler: POST vnf-to-node-map: ", v2nArray)
	if err := sfcplg.VNFToNodeMapCreate(v2nArray, true); err != nil {
		formatter.JSON(w, http.StatusBadRequest, struct{ Error string }{err.Error()})
		return
	}
	formatter.JSON(w, http.StatusOK, "OK")
}

// VNFServicesHandler GET: curl -v http://localhost:9191/sfc-controller/v2/config/vnf-services
func VNFServicesHandler(formatter *render.Render) http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		log.Debugf("VNF Services HTTP handler: Method %s, URL: %s", req.Method, req.URL)

		switch req.Method {
		case "GET":
			var array = make([]controller.VNFService, 0)
			for _, vss := range sfcplg.ramConfigCache.VNFServices {
				array = append(array, vss)
			}
			formatter.JSON(w, http.StatusOK, array)
		}
	}
}

// VNFServicesStatusHandler GET: curl -v http://localhost:9191/sfc-controller/v2/status/vnf-services
func VNFServicesStatusHandler(formatter *render.Render) http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		log.Debugf("VNF Services Status HTTP handler: Method %s, URL: %s", req.Method, req.URL)

		switch req.Method {
		case "GET":
			var array = make([]controller.VNFServiceState, 0)
			for _, vss := range sfcplg.ramConfigCache.VNFServicesState {
				array = append(array, *vss)
			}
			formatter.JSON(w, http.StatusOK, array)
		}
	}
}

// NodesStatusHandler GET: curl -v http://localhost:9191/sfc-controller/v2/status/nodes
func NodesStatusHandler(formatter *render.Render) http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		log.Debugf("Nodes Status HTTP handler: Method %s, URL: %s", req.Method, req.URL)

		switch req.Method {
		case "GET":
			var array = make([]controller.NodeState, 0)
			for _, n := range sfcplg.ramConfigCache.NodeState {
				array = append(array, *n)
			}
			formatter.JSON(w, http.StatusOK, array)
		}
	}
}

// NodesHandler GET: curl -v http://localhost:9191/sfc-controller/v2/config/nodes
func NodesHandler(formatter *render.Render) http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		log.Debugf("Nodes HTTP handler: Method %s, URL: %s", req.Method, req.URL)

		switch req.Method {
		case "GET":
			var array = make([]controller.Node, 0)
			for _, n := range sfcplg.ramConfigCache.Nodes {
				array = append(array, n)
			}
			formatter.JSON(w, http.StatusOK, array)
		}
	}
}

// InterfacesStateHandler GET: curl -v http://localhost:9191/sfc-controller/v2/status/interfaces
func InterfacesStateHandler(formatter *render.Render) http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		log.Debugf("Interfaces Status HTTP handler: Method %s, URL: %s", req.Method, req.URL)

		switch req.Method {
		case "GET":
			var array = make([]controller.InterfaceState, 0)
			for _, ifState := range sfcplg.ramConfigCache.InterfaceStates {
				array = append(array, ifState)
			}
			formatter.JSON(w, http.StatusOK, array)
		}
	}
}


// VNFServiceMeshesHandler GET: curl -v http://localhost:9191/sfc_controller/v2/config/vnf-service-meshes
func VNFServiceMeshesHandler(formatter *render.Render) http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		log.Debugf("VNF Service Meshes HTTP handler: Method %s, URL: %s", req.Method, req.URL)

		switch req.Method {
		case "GET":
			var array = make([]controller.VNFServiceMesh, 0)
			for _, n := range sfcplg.ramConfigCache.VNFServiceMeshes {
				array = append(array, n)
			}
			formatter.JSON(w, http.StatusOK, array)
		}
	}
}

// IPAMPoolsHandler GET: curl -v http://localhost:9191/sfc_controller/v2/config/ipam-pools
func IPAMPoolsHandler(formatter *render.Render) http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		log.Debugf("IPAM Pools HTTP handler: Method %s, URL: %s", req.Method, req.URL)

		switch req.Method {
		case "GET":
			var array = make([]controller.IPAMPool, 0)
			for _, ipamPool := range sfcplg.ramConfigCache.IPAMPools {
				array = append(array, ipamPool)
			}
			formatter.JSON(w, http.StatusOK, array)
		}
	}
}


// VPPEntriesHandler GET: curl -v http://localhost:9191/sfc_controller/v2/status/vpp-entries
func VPPEntriesHandler(formatter *render.Render) http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		log.Debugf("VPP entries HTTP handler: Method %s, URL: %s", req.Method, req.URL)

		switch req.Method {
		case "GET":
			var array = make([]vppagentapi.KeyValueEntryType, 0)
			for _, vpp := range sfcplg.ramConfigCache.VppEntries {
				array = append(array, *vpp)
			}
			formatter.JSON(w, http.StatusOK, array)
		}
	}
}

// curl -X GET http://localhost:9191/sfc_controller/api/v2/config/node/<entityName>
// curl -X POST -d '{"counter":30}' http://localhost:9191/sfc_controller/api/v2/config/node/<entityName>
// curl -X DELETE http://localhost:9191/sfc_controller/api/v2/config/node/<entityName>
func nodeHandler(formatter *render.Render) http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		log.Debugf("External Entity HTTP handler: Method %s, URL: %s", req.Method, req.URL)
		switch req.Method {
		case "GET":
			vars := mux.Vars(req)

			if n, exists := sfcplg.ramConfigCache.Nodes[vars[entityName]]; exists {
				formatter.JSON(w, http.StatusOK, n)
			} else {
				formatter.JSON(w, http.StatusNotFound, "node not found: "+vars[entityName])
			}
		case "POST":
			processNodePost(formatter, w, req)
		case "DELETE":
			processNodeDelete(formatter, w, req)
		}
	}
}

// create the node
func processNodePost(formatter *render.Render, w http.ResponseWriter, req *http.Request) {

	sfcplg.ConfigTransactionStart()
	defer sfcplg.ConfigTransactionEnd()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Debugf("Can't read body, error '%s'", err)
		formatter.JSON(w, http.StatusBadRequest, struct{ Error string }{err.Error()})
		return
	}

	var n controller.Node
	err = json.Unmarshal(body, &n)
	if err != nil {
		log.Debugf("Can't parse body, error '%s'", err)
		formatter.JSON(w, http.StatusBadRequest, struct{ Error string }{err.Error()})
		return
	}

	vars := mux.Vars(req)
	if vars[entityName] != n.Name {
		formatter.JSON(w, http.StatusBadRequest, "json name does not matach url name")
		return
	}

	if existing, exists := sfcplg.ramConfigCache.Nodes[vars[entityName]]; exists {
		if n.String() == existing.String() {
			formatter.JSON(w, http.StatusOK, "OK")
			return
		}
	}

	log.Debugf("processNodePost: POST node: ", n)
	if err := sfcplg.NodeCreate(&n, true); err != nil {
		formatter.JSON(w, http.StatusBadRequest, struct{ Error string }{err.Error()})
		return
	}

	formatter.JSON(w, http.StatusOK, "OK")
}

// create the node
func processNodeDelete(formatter *render.Render, w http.ResponseWriter, req *http.Request) {

	sfcplg.ConfigTransactionStart()
	defer sfcplg.ConfigTransactionEnd()

	vars := mux.Vars(req)
	if err := sfcplg.NodeDelete(vars[entityName]); err != nil {
		formatter.JSON(w, http.StatusBadRequest, struct{ Error string }{err.Error()})
		return
	}

	formatter.JSON(w, http.StatusOK, "OK")
}

// Example curl invocations: for obtaining the system parameters
//   - GET:  curl -X GET http://localhost:9191/sfc_controller/api/v2/SP
//   - POST: curl -v -X POST -d '{"mtu":1500}' http://localhost:9191/sfc_controller/api/v2/SP
func systemParametersHandler(formatter *render.Render) http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		log.Debugf("System Parameters HTTP handler: Method %s, URL: %s", req.Method, req.URL)
		switch req.Method {
		case "GET":

			formatter.JSON(w, http.StatusOK, sfcplg.ramConfigCache.SysParms)
			return
		case "POST":
			processSystemParametersPost(formatter, w, req)
		}
	}
}

// create the system parameters, replace the existing parms, and run side effects
func processSystemParametersPost(formatter *render.Render, w http.ResponseWriter, req *http.Request) {

	sfcplg.ConfigTransactionStart()
	defer sfcplg.ConfigTransactionEnd()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Debugf("Can't read body, error '%s'", err)
		formatter.JSON(w, http.StatusBadRequest, struct{ Error string }{err.Error()})
		return
	}
	var sp controller.SystemParameters
	err = json.Unmarshal(body, &sp)
	if err != nil {
		log.Debugf("Can't parse body, error '%s'", err)
		formatter.JSON(w, http.StatusBadRequest, struct{ Error string }{err.Error()})
		return
	}
	log.Debugf("processSystemParametersPost: POST system parameters: ", sp)
	if err := sfcplg.SysParmsCreate(&sp, true); err != nil {
		formatter.JSON(w, http.StatusBadRequest, struct{ Error string }{err.Error()})
		return
	}
	formatter.JSON(w, http.StatusOK, "OK")
}
