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

package ipam

import (
	"fmt"
	"net"

	"github.com/ligato/sfc-controller/plugins/controller/model"
)


// PoolAllocatorType data struct
type PoolAllocatorType struct {
	ipamPool *controller.IPAMPool
	ipv4Pool *Ipv4Pool
	//ipv6Pool *Ipv6Pool
}

func (p *PoolAllocatorType) String() string {
	str := fmt.Sprintf("IPAM Pool %s: addrs's: [%d-%d], %s",
		p.ipamPool.Name,
		p.ipamPool.StartRange,
		p.ipamPool.EndRange,
		p.ipv4Pool.String())
	return str
}

// SetAddress sets the address in the range as allocated
func (p *PoolAllocatorType) SetAddress(addrID uint32) error {
	if addrID < p.ipamPool.StartRange || addrID > p.ipamPool.EndRange {
		return fmt.Errorf("SetAddress: addr_id '%d' out of range '%d-%d",
			addrID, p.ipamPool.StartRange, p.ipamPool.EndRange)
	}
	_, err := p.ipv4Pool.SetIPInPool(addrID - p.ipamPool.StartRange + 1)
	if err != nil {
		return err
	}
	return nil
}

// AllocateIPAddress allocates a free ip address from the pool
func (p *PoolAllocatorType) AllocateIPAddress() (string, uint32, error) {
	if p.ipv4Pool != nil {
		return p.ipv4Pool.AllocateFromPool()
	}
	return "", 0, fmt.Errorf("AllocateIPAddress: %s is not ipv4 ... v6 not supported",
		p.ipamPool.Name)
}

// NewIPAMPoolAllocator allocates a ipam allocator
func NewIPAMPoolAllocator(ipamPool *controller.IPAMPool) *PoolAllocatorType {

	ip, network, err := net.ParseCIDR(ipamPool.Network)
	if err != nil {
		return nil
	}
	fmt.Printf("NewIPAMPoolAllocator: ip: %s, len(ip):%d, net: %s \n", ip, len(ip), network)
	fmt.Printf("NewIPAMPoolAllocator: len(ipTo4): %d, ipamPool: \n", len(ip.To4()), ipamPool)
	if len(ip.To4()) == net.IPv4len {
		poolAllocator := &PoolAllocatorType{
			ipamPool: ipamPool,
			ipv4Pool: NewIpv4Pool(ipamPool.Network, ipamPool.StartRange, ipamPool.EndRange),
		}
		return poolAllocator
	}

	return nil
}
