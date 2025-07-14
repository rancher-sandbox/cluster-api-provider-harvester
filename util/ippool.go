/*
Copyright 2025 SUSE.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"math/big"
	"net"
	"net/netip"

	"github.com/containernetworking/cni/pkg/types"
	cnip "github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/plugins/ipam/host-local/backend/allocator"
	lbv1beta1 "github.com/harvester/harvester-load-balancer/pkg/apis/loadbalancer.harvesterhci.io/v1beta1"
	"github.com/pkg/errors"
)

const (
	initialCapacity = 10
	p2pMaskStr      = "ffffffff"
)

// Store implements the backend.Store interface.
type Store struct {
	// IPPool is the pool of IP addresses.
	*lbv1beta1.IPPool
}

// NewStore creates a new store.
func NewStore(pool *lbv1beta1.IPPool) *Store {
	return &Store{
		IPPool: pool,
	}
}

// Lock locks the store.
func (s *Store) Lock() error {
	return nil
}

// Unlock unlocks the store.
func (s *Store) Unlock() error {
	return nil
}

// Close closes the store.
// It does nothing in this implementation.
func (s *Store) Close() error {
	return nil
}

// Reserve reserves the given IP address for the given range ID.
func (s *Store) Reserve(id, _ string, ip net.IP, _ string) (bool, error) {
	ipStr := ip.String()

	// return false if the IP is already reserved
	if s.Status.Allocated != nil {
		if _, ok := s.Status.Allocated[ipStr]; ok {
			return false, nil
		}
	}

	if s.Status.AllocatedHistory != nil {
		s.Status.AllocatedHistory[ipStr] = id
	}

	return true, nil
}

// LastReservedIP returns the last reserved IP address for the given range ID.
//
//nolint:revive
func (s *Store) LastReservedIP(rangeID string) (net.IP, error) {
	return net.ParseIP(s.Status.LastAllocated), nil
}

// Release releases the given IP address.
// If the IP is not allocated, it does nothing.
// If the IP is allocated, it removes it from the allocated list and increments the available count.
func (s *Store) Release(ip net.IP) error {
	if s.Status.Allocated == nil {
		return nil
	}

	ipStr := ip.String()

	if s.Status.AllocatedHistory == nil {
		s.Status.AllocatedHistory = make(map[string]string)
	}

	s.Status.AllocatedHistory[ipStr] = s.Status.Allocated[ipStr]
	delete(s.Status.Allocated, ipStr)

	s.Status.Available++

	return nil
}

// ReleaseByID releases all IP addresses allocated to the given ID.
// It removes them from the allocated list and increments the available count.
// It does nothing if the ID is not found in the allocated list.
func (s *Store) ReleaseByID(id string, _ string) error {
	if s.Status.Allocated == nil {
		return nil
	}

	for ip, applicant := range s.Status.Allocated {
		if applicant == id {
			if s.Status.AllocatedHistory == nil {
				s.Status.AllocatedHistory = make(map[string]string)
			}

			s.Status.AllocatedHistory[ip] = applicant
			delete(s.Status.Allocated, ip)

			s.Status.Available++
		}
	}

	return nil
}

// GetByID returns all IP addresses allocated to the given ID.
// It returns a slice of net.IP addresses.
// If no IP addresses are allocated to the ID, it returns an empty slice.
func (s *Store) GetByID(id string, _ string) []net.IP {
	ips := make([]net.IP, 0, initialCapacity)

	for ip, applicant := range s.Status.Allocated {
		if id == applicant {
			ips = append(ips, net.ParseIP(ip))
		}
	}

	return ips
}

// MakeRange converts a lbv1beta1.Range to an allocator.Range.
func MakeRange(r *lbv1beta1.Range) (*allocator.Range, error) {
	ip, ipNet, err := net.ParseCIDR(r.Subnet)
	if err != nil {
		return nil, errors.Errorf("invalide range %+v", r)
	}

	var defaultStart, defaultEnd, defaultGateway, start, end, gateway net.IP

	mask := ipNet.Mask.String()
	// If the subnet is a point to point IP
	if mask == p2pMaskStr {
		defaultStart = ip.To16()
		defaultEnd = ip.To16()
		defaultGateway = nil
	} else {
		// The rangeStart defaults to `.1` IP inside the `subnet` block.
		// The rangeEnd defaults to `.254` IP inside the `subnet` block for ipv4, `.255` for IPv6.
		// The gateway defaults to `.1` IP inside the `subnet` block.
		// Example:
		// 	  subnet: 192.168.0.0/24
		// 	  rangeStart: 192.168.0.1
		// 	  rangeEnd: 192.168.0.254
		// 	  gateway: 192.168.0.1
		// The gateway will be skipped during allocation.
		// To return the IP with 16 bytes representation as same as what the function net.ParseIP returns
		defaultStart = cnip.NextIP(ipNet.IP).To16()
		defaultEnd = lastIP(*ipNet).To16()
		defaultGateway = cnip.NextIP(ipNet.IP).To16()
	}

	start, err = parseIP(r.RangeStart, ipNet, defaultStart)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid range start %s", r.RangeStart)
	}

	end, err = parseIP(r.RangeEnd, ipNet, defaultEnd)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid range end %s", r.RangeEnd)
	}

	gateway, err = parseIP(r.Gateway, ipNet, defaultGateway)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid gateway %s", r.Gateway)
	}

	// Ensure start IP is smaller than end IP
	startAddr, _ := netip.AddrFromSlice(start)
	endAddr, _ := netip.AddrFromSlice(end)

	if startAddr.Compare(endAddr) > 0 {
		start, end = end, start
	}

	return &allocator.Range{
		RangeStart: start,
		RangeEnd:   end,
		Subnet:     types.IPNet(*ipNet),
		Gateway:    gateway,
	}, nil
}

func networkIP(n net.IPNet) net.IP {
	return n.IP.Mask(n.Mask)
}

func parseIP(ipStr string, ipNet *net.IPNet, defaultIP net.IP) (net.IP, error) {
	if ipStr == "" {
		return defaultIP, nil
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, errors.Errorf("invalid IP %s", ipStr)
	}

	if !ipNet.Contains(ip) {
		return nil, errors.Errorf("IP %s is out of subnet %s", ipStr, ipNet.String())
	}

	if ip.Equal(networkIP(*ipNet)) {
		return nil, errors.Errorf("IP %s is the network address", ipStr)
	}

	if ip.Equal(broadcastIP(*ipNet)) {
		return nil, errors.Errorf("IP %s is the broadcast address", ipStr)
	}

	return ip, nil
}

func broadcastIP(n net.IPNet) net.IP {
	broadcast := make(net.IP, len(n.IP))
	for i := range n.IP {
		broadcast[i] = n.IP[i] | ^n.Mask[i]
	}

	return broadcast
}

// Determine the last IP of a subnet, excluding the broadcast if IPv4.
func lastIP(subnet net.IPNet) net.IP {
	var end net.IP
	for i := range subnet.IP {
		end = append(end, subnet.IP[i]|^subnet.Mask[i])
	}

	if subnet.IP.To4() != nil {
		end[3]--
	}

	return end
}

// CountIP counts the number of IP addresses in the given range.
func CountIP(r *allocator.Range) int64 {
	count := big.NewInt(0).Add(big.NewInt(0).Sub(ipToInt(r.RangeEnd), ipToInt(r.RangeStart)), big.NewInt(1)).Int64()

	if r.Gateway != nil && r.Contains(r.Gateway) {
		count--
	}

	return count
}

func ipToInt(ip net.IP) *big.Int {
	if v := ip.To4(); v != nil {
		return big.NewInt(0).SetBytes(v)
	}

	return big.NewInt(0).SetBytes(ip.To16())
}
