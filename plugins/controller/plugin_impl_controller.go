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

	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/db/keyval"
	"github.com/ligato/cn-infra/db/keyval/etcdv3"
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/health/statuscheck"
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/cn-infra/logging/logrus"
	"github.com/ligato/cn-infra/rpc/rest"
	"github.com/ligato/sfc-controller/plugins/controller/idapi"
	"github.com/ligato/sfc-controller/plugins/controller/idapi/ipam"
	"github.com/ligato/sfc-controller/plugins/controller/model"
	"github.com/ligato/sfc-controller/plugins/controller/vppagentapi"
	"github.com/namsral/flag"
)

// PluginID is plugin identifier (must be unique throughout the system)
const PluginID core.PluginName = "SfcController"

var (
	sfcConfigFile     string // cli flag - see RegisterFlags
	cleanSfcDatastore bool   // cli flag - see RegisterFlags
	ContivKSREnabled bool   // cli flag - see RegisterFlags
	log               = logrus.DefaultLogger()
)

// RegisterFlags add command line flags.
func RegisterFlags() {
	flag.StringVar(&sfcConfigFile, "sfc-config", "",
		"Name of a sfc config (yaml) file to load at startup")
	flag.BoolVar(&cleanSfcDatastore, "clean", false,
		"Clean the SFC datastore entries")
	flag.BoolVar(&ContivKSREnabled, "contiv-ksr", false,
		"Interact with contiv ksr to learn k8s config/state")
}

// LogFlags dumps the command line flags
func LogFlags() {
	log.Debugf("LogFlags:")
	log.Debugf("\tsfcConfigFile:'%s'", sfcConfigFile)
	log.Debugf("\tclean:'%v'", cleanSfcDatastore)
	log.Debugf("\tcontiv ksr:'%v'", ContivKSREnabled)
}

// Init is the Go init() function for the s. It should
// contain the boiler plate initialization code that is executed
// when the plugin is loaded into the Agent.
func init() {
	// Logger must be initialized for each s individually.
	//log.SetLevel(logging.DebugLevel)
	log.SetLevel(logging.InfoLevel)

	RegisterFlags()
}

// CacheType is ram cache of controller entities indexed by entity name
type CacheType struct {
	// config
	Nodes            map[string]controller.Node
	VNFServices      map[string]controller.VNFService
	VNFToNodeMap     map[string]controller.VNFToNodeMap
	VNFServiceMeshes map[string]controller.VNFServiceMesh
	IPAMPools        map[string]controller.IPAMPool
	SysParms         controller.SystemParameters

	// state
	InterfaceStates              map[string]controller.InterfaceState
	VppEntries                   map[string]*vppagentapi.KeyValueEntryType
	VNFServicesState             map[string]*controller.VNFServiceState
	NodeState                    map[string]*controller.NodeState
	VNFToNodeStateMap            map[string]controller.VNFToNodeMap
	MacAddrAllocator             *idapi.MacAddrAllocatorType
	MemifIDAllocator             *idapi.MemifAllocatorType
	IPAMPoolAllocators           map[string]*ipam.PoolAllocatorType
	VNFServiceMeshVniAllocators  map[string]*idapi.VxlanVniAllocatorType
	VNFServiceMeshVxLanAddresses map[string]string
}

// Plugin contains the controllers information
type Plugin struct {
	Etcd    *etcdv3.Plugin
	HTTPmux *rest.Plugin
	*local.FlavorLocal
	ramConfigCache    CacheType
	configTransaction ConfigTransactionType
	db                keyval.ProtoBroker
}

// Init the controller, read the db, reconcile/resync, render config to etcd
func (s *Plugin) Init() error {

	log.Info("Init: enter ...", PluginID)
	defer log.Info("Init: exit ", PluginID)

	// Register providing status reports (push mode)
	s.StatusCheck.Register(PluginID, nil)
	s.StatusCheck.ReportStateChange(PluginID, statuscheck.Init, nil)

	s.db = s.Etcd.NewBroker(keyval.Root)
	s.InitRAMCache()

	//extentitydriver.SfcExternalEntityDriverInit()

	log.Infof("Initializing s '%s'", PluginID)

	// Flag variables registered in init() are ready to use in InitPlugin()
	LogFlags()

	// register northbound controller API's
	s.InitHTTPHandlers()

	if err := s.LoadSfcConfigIntoRAMCache(); err != nil {
		log.Error("error reading etcd config into ram cache: ", err)
		os.Exit(1)
	}

	if err := s.LoadVppAgentEntriesFromState(); err != nil {
		os.Exit(1)
	}

	// clean has to be here so we know which vpp agent keys to remove from the db
	if cleanSfcDatastore {
		s.CleanSfcDatastore()
		s.CleanVppAgentEntriesFromEtcd()
		s.InitRAMCache()
	}

	// If a startup yaml file is provided, then pull it into the ram cache and write it to the database
	// Note that there may already be already an existing database so the policy is that the config yaml
	// file will replace any conflicting entries in the database.
	if sfcConfigFile != "" {

		var yamlConfig *YamlConfig
		var err error

		if yamlConfig, err = s.ReadYamlConfigFromFile(sfcConfigFile); err != nil {
			log.Error("error loading config: ", err)
			os.Exit(1)
		}

		if err := s.ProcessYamlConfig(yamlConfig); err != nil {
			log.Error("error copying config: ", err)
			os.Exit(1)
		}
	}

	log.Infof("Init: controller cache: %v", s.ramConfigCache)

	return nil
}

// AfterInit is called after all plugin are init-ed
func (s *Plugin) AfterInit() error {
	log.Info("AfterInit:", PluginID)

	// at this point, plugins are all laoded, all is read in from the database
	// so render the config ... note: resync will ensure etcd is written to
	// unnessarily

	s.ConfigTransactionStart()
	if err := s.RenderConfig(); err != nil {
		os.Exit(1)
	}
	s.ConfigTransactionEnd()

	s.StartWatchers()

	s.StatusCheck.ReportStateChange(PluginID, statuscheck.OK, nil)

	return nil
}

// InitRAMCache creates the ram cache
func (s *Plugin) InitRAMCache() {

	// setting entries to nil will free any old entries if there were any
	s.ramConfigCache.Nodes = nil
	s.ramConfigCache.Nodes = make(map[string]controller.Node)

	s.ramConfigCache.InterfaceStates = nil
	s.ramConfigCache.InterfaceStates = make(map[string]controller.InterfaceState)

	s.ramConfigCache.VNFServices = nil
	s.ramConfigCache.VNFServices = make(map[string]controller.VNFService)

	s.ramConfigCache.VNFServiceMeshes = nil
	s.ramConfigCache.VNFServiceMeshes = make(map[string]controller.VNFServiceMesh)

	s.ramConfigCache.IPAMPools = nil
	s.ramConfigCache.IPAMPools = make(map[string]controller.IPAMPool)
	s.ramConfigCache.IPAMPoolAllocators = nil
	s.ramConfigCache.IPAMPoolAllocators = make(map[string]*ipam.PoolAllocatorType)

	s.ramConfigCache.VNFServiceMeshVniAllocators = nil
	s.ramConfigCache.VNFServiceMeshVniAllocators = make(map[string]*idapi.VxlanVniAllocatorType)

	s.ramConfigCache.VNFServiceMeshVxLanAddresses = nil
	s.ramConfigCache.VNFServiceMeshVxLanAddresses = make(map[string]string)

	s.ramConfigCache.VNFToNodeMap = nil
	s.ramConfigCache.VNFToNodeMap = make(map[string]controller.VNFToNodeMap)

	s.ramConfigCache.VNFToNodeStateMap = nil
	s.ramConfigCache.VNFToNodeStateMap = make(map[string]controller.VNFToNodeMap)

	s.ramConfigCache.VNFServicesState = nil
	s.ramConfigCache.VNFServicesState = make(map[string]*controller.VNFServiceState)

	s.ramConfigCache.NodeState = nil
	s.ramConfigCache.NodeState = make(map[string]*controller.NodeState)

	s.ramConfigCache.VppEntries = nil
	s.ramConfigCache.VppEntries = make(map[string]*vppagentapi.KeyValueEntryType)

	s.ramConfigCache.MacAddrAllocator = nil
	s.ramConfigCache.MacAddrAllocator = idapi.NewMacAddrAllocator()

	s.ramConfigCache.MemifIDAllocator = nil
	s.ramConfigCache.MemifIDAllocator = idapi.NewMemifAllocator()
}

// Close performs close down procedures
func (s *Plugin) Close() error {
	//return safeclose.Close(extentitydriver.EEOperationChannel)
	return nil
}

// RenderConfig runs though the cache and renders the config
func (s *Plugin) RenderConfig() error {
	if err := s.NodesRender(); err != nil {
		return err
	}
	if err := s.VNFServicesRender(); err != nil {
		return err
	}
	return nil
}
