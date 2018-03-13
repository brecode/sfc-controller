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
	"sync"

	"github.com/ligato/sfc-controller/plugins/controller/model"
	"github.com/ligato/sfc-controller/plugins/controller/vppagentapi"
)

// ConfigTransactionType is a cache of before and after entries per
// transaction type where a transaction could be a nb_api rest request,
// or a complete system resync which is initiated at startup after the
// db has been read in, or a loss of comms with etcd has occured.
// Note: NO nb_api's are accepted during transaction processing as they
// are atomic operations and are mutex protected during each transaciton.
type ConfigTransactionType struct {
	beforeEntriesMap      map[string]*vppagentapi.KeyValueEntryType
	afterEntriesMap       map[string]*vppagentapi.KeyValueEntryType
	transactionInProgress bool
}

var configMutex sync.Mutex

// ConfigTransactionStart initates state for starting a config transaction
func (s *Plugin) ConfigTransactionStart() {

	configMutex.Lock()

	log.Info("ConfigTransactionStart: starting ...")
	defer log.Info("ConfigTransactionStart: finished ...")

	s.configTransaction.beforeEntriesMap = nil
	s.configTransaction.afterEntriesMap = nil

	s.configTransaction.beforeEntriesMap = make(map[string]*vppagentapi.KeyValueEntryType)
	s.configTransaction.afterEntriesMap = make(map[string]*vppagentapi.KeyValueEntryType)
}

// ConfigCleanupErrorOcurredDuringRendering removes the rendered entries
func (s *Plugin) ConfigCleanupErrorOcurredDuringRendering() {

	// by removing the after entries, the end of transaction processing below will
	// be able to cleanup properly
	s.configTransaction.afterEntriesMap = nil
	s.configTransaction.afterEntriesMap = make(map[string]*vppagentapi.KeyValueEntryType)
}

// ConfigTransactionEnd traverese new and old and updates etcd
func (s *Plugin) ConfigTransactionEnd() error {

	// The transaction consists of all the types of objects that are created as
	// a result of processing the nodes, and services.  These are interfaces,
	// bridged domains, and static routes, etc.
	// When a transaction starts, we copy all these and ensure all existing vpp
	// entries are stored in the "before" cache. Then as the configuration is
	// processed, the "new" objects are added to the transaction "after" cache.
	// When the configuration is processed, a post processing of the before and
	// after caches is performed.
	// All entries in the before cache are processed one-by-one.  If the before
	// entry is not in the after cache, then clearly it is not needed and removed
	// from ETCD.  If it is in the after cache, then there are two cases.
	// If the entries match, it is removed from the after cache and ETCD is not
	// "touched".  If the entries are different, it remains in the after cache
	// and awaits transaction end "after" cache post processing.  Once all the
	// before entries have been processed, the after cache is processed.
	// If there are still entries in this cache, they are all written to ETCD.

	// The reason for this transactional approach is as follows: some ETCD entries
	// will be added and updated multiple times during processing of the
	// configuration and there is NO sense continually changing ETCD for an entry
	// until it is fully modified by the configuration processing.  This is why
	// an "after" cache is maintained.  Then post processing will ensure the
	// "final" values of an object are written only ONCE to the ETCD cache.
	// An example of this is bridge domains.  Initially for a host, a default
	// east-west BD is added to the system, then as interfaces are associated
	// with the BD, the BD is updated.  If we tried to continually update the
	// ETCD entry for this BD as we went along, we would improperly set the BD
	// to interim configs until it all the config is performed and the BD reaches
	// its final config.  This would have bad effects on the vpp-agents
	// as they would be forced to react to each BD change and data flow would
	// be affected.  The goal of the reconcile resync is to ONLY make changes
	// if there are new and/or obselete configs.  Existing configs should
	// reamin un-affected by the transaction process.

	defer configMutex.Unlock()

	log.Info("ConfigTransactionEnd: starting ...")
	defer log.Info("ConfigTransactionEnd: finished ...")

	// traverse the entries in the before cache
	for key := range s.configTransaction.beforeEntriesMap {
		before := s.configTransaction.beforeEntriesMap[key]
		after, existsInAfterCache := s.configTransaction.afterEntriesMap[key]
		if !existsInAfterCache {
			exists, err := s.db.Delete(key)          // remove from etcd
			delete(s.ramConfigCache.VppEntries, key) // remove from sys ram cache
			log.Info("ConfigTransactionEnd: remove key from etcd and system cache: ", key, exists, err)
			log.Info("ConfigTransactionEnd: remove before entry: ", before)
			delete(s.configTransaction.beforeEntriesMap, key)
		} else {
			if before.Equal(after) {
				delete(s.configTransaction.afterEntriesMap, key) // ensure dont resend to etcd
			} else {
				log.Info("ConfigTransactionEnd: before != after ... before: ", before)
				log.Info("ConfigTransactionEnd: before != after ... after: ", after)
			}
		}
	}
	// now post process the after cache, write the remaining entries to etcd
	for key, after := range s.configTransaction.afterEntriesMap {
		log.Info("ConfigTransactionEnd: add key to etcd: ", key, after)
		err := after.WriteToEtcd(s.db)
		if err != nil {
			return err
		}
	}

	s.transferAfterVppKVEntriesToSystemCache()

	return nil
}

// ConfigTransactionAddVppEntries caches the new entry in the transaction new/after map
func (s *Plugin) ConfigTransactionAddVppEntries(
	renderedVppAgentEntries []*controller.RenderedVppAgentEntry,
	vppKVs []*vppagentapi.KeyValueEntryType) (newArray []*controller.RenderedVppAgentEntry) {
	newArray = renderedVppAgentEntries
	for _, vppKV := range vppKVs {
		newArray = s.ConfigTransactionAddVppEntry(newArray, vppKV)
	}
	return newArray
}

// ConfigTransactionAddVppEntry caches the new entry in the transaction new/after map
func (s *Plugin) ConfigTransactionAddVppEntry(
	renderedVppAgentEntries []*controller.RenderedVppAgentEntry,
	vppKV *vppagentapi.KeyValueEntryType) (newArray []*controller.RenderedVppAgentEntry) {

	if _, exists := s.configTransaction.afterEntriesMap[vppKV.VppKey]; !exists {
		// initialize a new rendered vpp agent entry and append it to the array
		renderedVppEntry := &controller.RenderedVppAgentEntry{
			VppAgentKey:  vppKV.VppKey,
			VppAgentType: vppKV.VppEntryType,
		}
		newArray = append(renderedVppAgentEntries, renderedVppEntry)
	} else {
		newArray = renderedVppAgentEntries
	}
	// add the new or existing kv entry to the config transaction after map
	s.configTransaction.afterEntriesMap[vppKV.VppKey] = vppKV

	log.Debugf("ConfigTransactionAddVppEntry: rendered array len: %d, kv:%v, ",
		len(newArray), vppKV)

	return newArray
}

// CopyRenderedVppAgentEntriesToBeforeConfigTransaction cache the existing set before new keys are rendered
func (s *Plugin) CopyRenderedVppAgentEntriesToBeforeConfigTransaction(
	vppAgentEntries []*controller.RenderedVppAgentEntry) {

	for _, vppAgentEntry := range vppAgentEntries {
		log.Debugf("CopyRendered...BeforeConfigTransaction: entry=%v", vppAgentEntry)
		if vppKV, exists := s.ramConfigCache.VppEntries[vppAgentEntry.VppAgentKey]; !exists {
			log.Warnf("CopyRendered...BeforeConfigTransaction: ouch ... missing vpp cache entry: %s",
				vppAgentEntry.VppAgentKey)
			vppKV = &vppagentapi.KeyValueEntryType{
				VppKey:       vppAgentEntry.VppAgentKey,
				VppEntryType: vppAgentEntry.VppAgentType,
			}
			s.configTransaction.beforeEntriesMap[vppAgentEntry.VppAgentKey] = vppKV
		} else {
			s.configTransaction.beforeEntriesMap[vppAgentEntry.VppAgentKey] = vppKV
		}
	}
	// log.Debugf("CopyRenderedVppAgentEntriesToBeforeConfigTransaction: beforeMap: %v",
	// 	s.configTransaction.beforeEntriesMap)
}

// transferAfterVppKVEntriesToSystemCache updates the ssytem cache with the new vpp agent entries
func (s *Plugin) transferAfterVppKVEntriesToSystemCache() {

	for _, vppKV := range s.configTransaction.afterEntriesMap {
		s.ramConfigCache.VppEntries[vppKV.VppKey] = vppKV
	}
}

// LoadVppAgentEntriesFromState uses key/type from state to lad vpp entries from etcd
func (s *Plugin) LoadVppAgentEntriesFromState() error {

	log.Debugf("LoadVppAgentEntriesFromState: processing vnf services state: num: %d",
		len(s.ramConfigCache.VNFServicesState))
	for _, vs := range s.ramConfigCache.VNFServicesState {
		log.Debugf("LoadVppAgentEntriesFromState: processing vnf service state: %s", vs.Name)
		if err := s.LoadVppAgentEntriesFromRenderedVppAgentEntries(vs.RenderedVppAgentEntries); err != nil {
			return err
		}
	}
	log.Debugf("LoadVppAgentEntriesFromState: processing nodes state: num: %d",
		len(s.ramConfigCache.NodeState))
	for _, ns := range s.ramConfigCache.NodeState {
		log.Debugf("LoadVppAgentEntriesFromState: processing node state: %s", ns.Name)
		if err := s.LoadVppAgentEntriesFromRenderedVppAgentEntries(ns.RenderedVppAgentEntries); err != nil {
			return err
		}
	}

	return nil
}

// LoadVppAgentEntriesFromRenderedVppAgentEntries load from etcd
func (s *Plugin) LoadVppAgentEntriesFromRenderedVppAgentEntries(
	vppAgentEntries []*controller.RenderedVppAgentEntry) error {

	log.Debugf("LoadVppAgentEntriesFromRenderedVppAgentEntries: num: %d, %v",
		len(vppAgentEntries), vppAgentEntries)
	for _, vppAgentEntry := range vppAgentEntries {

		vppKVEntry := vppagentapi.NewKVEntry(vppAgentEntry.VppAgentKey, vppAgentEntry.VppAgentType)
		found, err := vppKVEntry.ReadFromEtcd(s.db)
		if err != nil {
			return err
		}
		if found {
			s.ramConfigCache.VppEntries[vppKVEntry.VppKey] = vppKVEntry
		}
	}

	return nil
}

// CleanVppAgentEntriesFromEtcd load from etcd
func (s *Plugin) CleanVppAgentEntriesFromEtcd() {
	log.Debugf("CleanVppAgentEntriesFromEtcd: removing all vpp keys managed by the controller")
	for _, kvEntry := range s.ramConfigCache.VppEntries {
		s.DeleteFromDatastore(kvEntry.VppKey)
	}
}
