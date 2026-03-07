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

package controller

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	kubevirtv1 "kubevirt.io/api/core/v1"

	"github.com/rancher-sandbox/cluster-api-provider-harvester/api/v1alpha1"
)

var _ = Describe("Convert HarvesterMachine networks to Kubevirt Networks", func() {
	var hvMachineNetworks *v1alpha1.HarvesterMachine

	var kvNetworks []kubevirtv1.Network

	BeforeEach(func() {
		hvMachineNetworks = &v1alpha1.HarvesterMachine{
			Spec: v1alpha1.HarvesterMachineSpec{
				Networks: []string{"network1", "network2"},
			},
		}

		kvNetworks = []kubevirtv1.Network{
			{
				Name: "nic-1",
				NetworkSource: kubevirtv1.NetworkSource{
					Multus: &kubevirtv1.MultusNetwork{
						NetworkName: "network1",
					},
				},
			},
			{
				Name: "nic-2",
				NetworkSource: kubevirtv1.NetworkSource{
					Multus: &kubevirtv1.MultusNetwork{
						NetworkName: "network2",
					},
				},
			},
		}
	})
	Context("When we provide a list of HarvesterMachine networks", func() {
		It("Should return a list of Kubevirt Networks", func() {
			Expect(getKubevirtNetworksFromHarvesterMachine(hvMachineNetworks)).To(Equal(kvNetworks))
		})
	})
})

var _ = Describe("buildNetworkData", func() {
	Context("DHCP with 1 NIC (no NetworkConfig)", func() {
		It("Should generate DHCP config for eth0", func() {
			scope := &Scope{
				HarvesterMachine: &v1alpha1.HarvesterMachine{
					Spec: v1alpha1.HarvesterMachineSpec{
						Networks: []string{"default/production"},
					},
				},
				HarvesterCluster: &v1alpha1.HarvesterCluster{},
			}

			result := buildNetworkData(scope)
			Expect(result).To(ContainSubstring("version: 1"))
			Expect(result).To(ContainSubstring("name: eth0"))
			Expect(result).To(ContainSubstring("type: dhcp"))
			Expect(result).NotTo(ContainSubstring("type: static"))
			Expect(result).NotTo(ContainSubstring("eth1"))
		})
	})

	Context("DHCP with 2 NICs (no NetworkConfig)", func() {
		It("Should generate DHCP config for eth0 and eth1", func() {
			scope := &Scope{
				HarvesterMachine: &v1alpha1.HarvesterMachine{
					Spec: v1alpha1.HarvesterMachineSpec{
						Networks: []string{"default/production", "default/management"},
					},
				},
				HarvesterCluster: &v1alpha1.HarvesterCluster{},
			}

			result := buildNetworkData(scope)
			Expect(result).To(ContainSubstring("name: eth0"))
			Expect(result).To(ContainSubstring("name: eth1"))
			Expect(strings.Count(result, "type: dhcp")).To(Equal(2))
			Expect(result).NotTo(ContainSubstring("type: static"))
		})
	})

	Context("Static eth0 + DHCP eth1 (with NetworkConfig, 2 NICs)", func() {
		It("Should generate static config for eth0 and DHCP for eth1", func() {
			scope := &Scope{
				HarvesterMachine: &v1alpha1.HarvesterMachine{
					Spec: v1alpha1.HarvesterMachineSpec{
						Networks: []string{"default/production", "default/management"},
					},
				},
				HarvesterCluster: &v1alpha1.HarvesterCluster{
					Spec: v1alpha1.HarvesterClusterSpec{
						VMNetworkConfig: &v1alpha1.VMNetworkConfig{
							SubnetMask: "255.255.0.0",
							Gateway:    "172.16.0.1",
							DNSServers: []string{"172.16.0.1"},
						},
					},
				},
				EffectiveNetworkConfig: &v1alpha1.NetworkConfig{
					Address:    "172.16.3.42",
					Gateway:    "172.16.0.1",
					DNSServers: []string{"172.16.0.1"},
				},
			}

			result := buildNetworkData(scope)
			Expect(result).To(ContainSubstring("name: eth0"))
			Expect(result).To(ContainSubstring("type: static"))
			Expect(result).To(ContainSubstring("address: 172.16.3.42"))
			Expect(result).To(ContainSubstring("netmask: 255.255.0.0"))
			Expect(result).To(ContainSubstring("gateway: 172.16.0.1"))
			Expect(result).To(ContainSubstring("name: eth1"))
			Expect(result).To(ContainSubstring("type: dhcp"))
			Expect(result).To(ContainSubstring("type: nameserver"))
			Expect(result).To(ContainSubstring("- 172.16.0.1"))
		})
	})

	Context("Static eth0 with 1 NIC (regression test)", func() {
		It("Should generate the same output as the old inline code", func() {
			scope := &Scope{
				HarvesterMachine: &v1alpha1.HarvesterMachine{
					Spec: v1alpha1.HarvesterMachineSpec{
						Networks: []string{"default/production"},
					},
				},
				HarvesterCluster: &v1alpha1.HarvesterCluster{
					Spec: v1alpha1.HarvesterClusterSpec{
						VMNetworkConfig: &v1alpha1.VMNetworkConfig{
							SubnetMask: "255.255.0.0",
							Gateway:    "172.16.0.1",
							DNSServers: []string{"172.16.0.1"},
						},
					},
				},
				EffectiveNetworkConfig: &v1alpha1.NetworkConfig{
					Address:    "172.16.3.40",
					Gateway:    "172.16.0.1",
					DNSServers: []string{"172.16.0.1"},
				},
			}

			expected := `version: 1
config:
  - type: physical
    name: eth0
    subnets:
      - type: static
        address: 172.16.3.40
        netmask: 255.255.0.0
        gateway: 172.16.0.1
  - type: nameserver
    address:
      - 172.16.0.1
`
			result := buildNetworkData(scope)
			Expect(result).To(Equal(expected))
		})
	})
})
