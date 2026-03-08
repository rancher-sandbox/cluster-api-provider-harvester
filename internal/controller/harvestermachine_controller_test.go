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
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kubevirtv1 "kubevirt.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1 "github.com/rancher-sandbox/cluster-api-provider-harvester/api/v1alpha1"
)

// =============================================================================
// Tests for getKubevirtNetworksFromHarvesterMachine (existing)
// =============================================================================

var _ = Describe("Convert HarvesterMachine networks to Kubevirt Networks", func() {
	var hvMachineNetworks *infrav1.HarvesterMachine

	var kvNetworks []kubevirtv1.Network

	BeforeEach(func() {
		hvMachineNetworks = &infrav1.HarvesterMachine{
			Spec: infrav1.HarvesterMachineSpec{
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

// =============================================================================
// Tests for buildNetworkDataStatic and buildDHCPCloudInit (existing + new)
// =============================================================================

var _ = Describe("buildNetworkData", func() {
	Context("DHCP cloud-init with 1 NIC", func() {
		It("Should generate write_files and bootcmd for dhclient", func() {
			scope := &Scope{
				HarvesterMachine: &infrav1.HarvesterMachine{
					Spec: infrav1.HarvesterMachineSpec{
						Networks: []string{"default/production"},
					},
				},
				HarvesterCluster: &infrav1.HarvesterCluster{},
			}

			result := buildDHCPCloudInit(scope)
			Expect(result).To(ContainSubstring("bootcmd:"))
			Expect(result).To(ContainSubstring("dhclient-script-caphv.sh"))
			Expect(result).To(ContainSubstring("dhclient"))
			Expect(result).To(ContainSubstring("eth0"))
			Expect(result).To(ContainSubstring("cat >"))                         // script created inline in bootcmd
			Expect(strings.Count(result, "dhclient")).To(BeNumerically(">=", 2)) // script ref + bootcmd
		})
	})

	Context("DHCP cloud-init with 2 NICs", func() {
		It("Should generate bootcmd entries for both interfaces", func() {
			scope := &Scope{
				HarvesterMachine: &infrav1.HarvesterMachine{
					Spec: infrav1.HarvesterMachineSpec{
						Networks: []string{"default/production", "default/management"},
					},
				},
				HarvesterCluster: &infrav1.HarvesterCluster{},
			}

			result := buildDHCPCloudInit(scope)
			Expect(result).To(ContainSubstring("eth0"))
			Expect(result).To(ContainSubstring("eth1"))
			Expect(result).To(ContainSubstring("dhclient-eth0.lease"))
			Expect(result).To(ContainSubstring("dhclient-eth1.lease"))
		})
	})

	Context("Static eth0 + DHCP eth1 (with NetworkConfig, 2 NICs)", func() {
		It("Should generate static config for eth0 and DHCP for eth1", func() {
			scope := &Scope{
				HarvesterMachine: &infrav1.HarvesterMachine{
					Spec: infrav1.HarvesterMachineSpec{
						Networks: []string{"default/production", "default/management"},
					},
				},
				HarvesterCluster: &infrav1.HarvesterCluster{
					Spec: infrav1.HarvesterClusterSpec{
						VMNetworkConfig: &infrav1.VMNetworkConfig{
							SubnetMask: "255.255.0.0",
							Gateway:    "172.16.0.1",
							DNSServers: []string{"172.16.0.1"},
						},
					},
				},
				EffectiveNetworkConfig: &infrav1.NetworkConfig{
					Address:    "172.16.3.42",
					Gateway:    "172.16.0.1",
					DNSServers: []string{"172.16.0.1"},
				},
			}

			result := buildNetworkDataStatic(scope)
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
				HarvesterMachine: &infrav1.HarvesterMachine{
					Spec: infrav1.HarvesterMachineSpec{
						Networks: []string{"default/production"},
					},
				},
				HarvesterCluster: &infrav1.HarvesterCluster{
					Spec: infrav1.HarvesterClusterSpec{
						VMNetworkConfig: &infrav1.VMNetworkConfig{
							SubnetMask: "255.255.0.0",
							Gateway:    "172.16.0.1",
							DNSServers: []string{"172.16.0.1"},
						},
					},
				},
				EffectiveNetworkConfig: &infrav1.NetworkConfig{
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
			result := buildNetworkDataStatic(scope)
			Expect(result).To(Equal(expected))
		})
	})

	Context("Static eth0 with DNS search domains", func() {
		It("Should include search domains in nameserver config", func() {
			scope := &Scope{
				HarvesterMachine: &infrav1.HarvesterMachine{
					Spec: infrav1.HarvesterMachineSpec{
						Networks: []string{"default/production"},
					},
				},
				HarvesterCluster: &infrav1.HarvesterCluster{
					Spec: infrav1.HarvesterClusterSpec{
						VMNetworkConfig: &infrav1.VMNetworkConfig{
							SubnetMask: "255.255.0.0",
							Gateway:    "172.16.0.1",
							DNSServers: []string{"172.16.0.1"},
						},
					},
				},
				EffectiveNetworkConfig: &infrav1.NetworkConfig{
					Address:    "172.16.3.42",
					Gateway:    "172.16.0.1",
					DNSServers: []string{"172.16.0.1"},
					DNSSearch:  []string{"home.lo", "cluster.local"},
				},
			}
			result := buildNetworkDataStatic(scope)
			Expect(result).To(ContainSubstring("search:"))
			Expect(result).To(ContainSubstring("- home.lo"))
			Expect(result).To(ContainSubstring("- cluster.local"))
			Expect(result).To(ContainSubstring("type: nameserver"))
			Expect(result).To(ContainSubstring("- 172.16.0.1"))
		})
	})

	Context("Static eth0 with multiple DNS servers", func() {
		It("Should list all DNS servers", func() {
			scope := &Scope{
				HarvesterMachine: &infrav1.HarvesterMachine{
					Spec: infrav1.HarvesterMachineSpec{
						Networks: []string{"default/production"},
					},
				},
				HarvesterCluster: &infrav1.HarvesterCluster{
					Spec: infrav1.HarvesterClusterSpec{
						VMNetworkConfig: &infrav1.VMNetworkConfig{
							SubnetMask: "255.255.255.0",
							Gateway:    "10.0.0.1",
							DNSServers: []string{"8.8.8.8", "8.8.4.4"},
						},
					},
				},
				EffectiveNetworkConfig: &infrav1.NetworkConfig{
					Address:    "10.0.0.50",
					Gateway:    "10.0.0.1",
					DNSServers: []string{"8.8.8.8", "8.8.4.4"},
				},
			}

			result := buildNetworkDataStatic(scope)
			Expect(result).To(ContainSubstring("- 8.8.8.8"))
			Expect(result).To(ContainSubstring("- 8.8.4.4"))
			Expect(result).To(ContainSubstring("netmask: 255.255.255.0"))
		})
	})

	Context("Static with no DNS servers", func() {
		It("Should not include nameserver section", func() {
			scope := &Scope{
				HarvesterMachine: &infrav1.HarvesterMachine{
					Spec: infrav1.HarvesterMachineSpec{
						Networks: []string{"default/production"},
					},
				},
				HarvesterCluster: &infrav1.HarvesterCluster{
					Spec: infrav1.HarvesterClusterSpec{
						VMNetworkConfig: &infrav1.VMNetworkConfig{
							SubnetMask: "255.255.0.0",
							Gateway:    "172.16.0.1",
						},
					},
				},
				EffectiveNetworkConfig: &infrav1.NetworkConfig{
					Address:    "172.16.3.40",
					Gateway:    "172.16.0.1",
					DNSServers: []string{},
				},
			}

			result := buildNetworkDataStatic(scope)
			Expect(result).NotTo(ContainSubstring("type: nameserver"))
		})
	})

	Context("Static with VMNetworkConfig nil but default subnet mask", func() {
		It("Should use default subnet mask 255.255.0.0", func() {
			scope := &Scope{
				HarvesterMachine: &infrav1.HarvesterMachine{
					Spec: infrav1.HarvesterMachineSpec{
						Networks: []string{"default/production"},
					},
				},
				HarvesterCluster: &infrav1.HarvesterCluster{},
				EffectiveNetworkConfig: &infrav1.NetworkConfig{
					Address:    "172.16.3.40",
					Gateway:    "172.16.0.1",
					DNSServers: []string{},
				},
			}

			result := buildNetworkDataStatic(scope)
			Expect(result).To(ContainSubstring("netmask: 255.255.0.0"))
		})
	})

	Context("Static with 3 NICs", func() {
		It("Should create static for eth0 and DHCP for eth1 and eth2", func() {
			scope := &Scope{
				HarvesterMachine: &infrav1.HarvesterMachine{
					Spec: infrav1.HarvesterMachineSpec{
						Networks: []string{"default/prod", "default/mgmt", "default/storage"},
					},
				},
				HarvesterCluster: &infrav1.HarvesterCluster{
					Spec: infrav1.HarvesterClusterSpec{
						VMNetworkConfig: &infrav1.VMNetworkConfig{
							SubnetMask: "255.255.0.0",
							Gateway:    "172.16.0.1",
							DNSServers: []string{"172.16.0.1"},
						},
					},
				},
				EffectiveNetworkConfig: &infrav1.NetworkConfig{
					Address:    "172.16.3.40",
					Gateway:    "172.16.0.1",
					DNSServers: []string{"172.16.0.1"},
				},
			}

			result := buildNetworkDataStatic(scope)
			Expect(result).To(ContainSubstring("name: eth0"))
			Expect(result).To(ContainSubstring("name: eth1"))
			Expect(result).To(ContainSubstring("name: eth2"))
			// eth0 should be static
			Expect(result).To(ContainSubstring("type: static"))
			// eth1 and eth2 should be DHCP
			Expect(strings.Count(result, "type: dhcp")).To(Equal(2))
		})
	})

	Context("DHCP cloud-init with 3 NICs", func() {
		It("Should generate bootcmd entries for all interfaces", func() {
			scope := &Scope{
				HarvesterMachine: &infrav1.HarvesterMachine{
					Spec: infrav1.HarvesterMachineSpec{
						Networks: []string{"default/prod", "default/mgmt", "default/storage"},
					},
				},
				HarvesterCluster: &infrav1.HarvesterCluster{},
			}

			result := buildDHCPCloudInit(scope)
			Expect(result).To(ContainSubstring("eth0"))
			Expect(result).To(ContainSubstring("eth1"))
			Expect(result).To(ContainSubstring("eth2"))
			Expect(result).To(ContainSubstring("dhclient-eth0.lease"))
			Expect(result).To(ContainSubstring("dhclient-eth1.lease"))
			Expect(result).To(ContainSubstring("dhclient-eth2.lease"))
		})
	})
})

// =============================================================================
// Tests for isVMRunning
// =============================================================================

var _ = Describe("isVMRunning", func() {
	It("should return true for RunStrategyAlways", func() {
		strategy := kubevirtv1.RunStrategyAlways
		vm := &kubevirtv1.VirtualMachine{
			Spec: kubevirtv1.VirtualMachineSpec{RunStrategy: &strategy},
		}
		Expect(isVMRunning(vm)).To(BeTrue())
	})

	It("should return true for RunStrategyRerunOnFailure", func() {
		strategy := kubevirtv1.RunStrategyRerunOnFailure
		vm := &kubevirtv1.VirtualMachine{
			Spec: kubevirtv1.VirtualMachineSpec{RunStrategy: &strategy},
		}
		Expect(isVMRunning(vm)).To(BeTrue())
	})

	It("should return true for RunStrategyOnce", func() {
		strategy := kubevirtv1.RunStrategyOnce
		vm := &kubevirtv1.VirtualMachine{
			Spec: kubevirtv1.VirtualMachineSpec{RunStrategy: &strategy},
		}
		Expect(isVMRunning(vm)).To(BeTrue())
	})

	It("should return false for RunStrategyHalted", func() {
		strategy := kubevirtv1.RunStrategyHalted
		vm := &kubevirtv1.VirtualMachine{
			Spec: kubevirtv1.VirtualMachineSpec{RunStrategy: &strategy},
		}
		Expect(isVMRunning(vm)).To(BeFalse())
	})

	It("should fallback to status.ready when both running and runStrategy are set", func() {
		strategy := kubevirtv1.RunStrategyAlways
		running := true
		vm := &kubevirtv1.VirtualMachine{
			Spec: kubevirtv1.VirtualMachineSpec{
				RunStrategy: &strategy,
				Running:     &running,
			},
			Status: kubevirtv1.VirtualMachineStatus{Ready: true},
		}
		Expect(isVMRunning(vm)).To(BeTrue())
	})

	It("should return false when both running and runStrategy set and status not ready", func() {
		strategy := kubevirtv1.RunStrategyAlways
		running := true
		vm := &kubevirtv1.VirtualMachine{
			Spec: kubevirtv1.VirtualMachineSpec{
				RunStrategy: &strategy,
				Running:     &running,
			},
			Status: kubevirtv1.VirtualMachineStatus{Ready: false},
		}
		Expect(isVMRunning(vm)).To(BeFalse())
	})

	It("should return true when only spec.running is true (deprecated path)", func() {
		running := true
		vm := &kubevirtv1.VirtualMachine{
			Spec: kubevirtv1.VirtualMachineSpec{Running: &running},
		}
		// When only running is set, RunStrategy() returns RunStrategyAlways for running=true
		Expect(isVMRunning(vm)).To(BeTrue())
	})

	It("should return false when only spec.running is false (deprecated path)", func() {
		running := false
		vm := &kubevirtv1.VirtualMachine{
			Spec: kubevirtv1.VirtualMachineSpec{Running: &running},
		}
		// When only running is set, RunStrategy() returns RunStrategyHalted for running=false
		Expect(isVMRunning(vm)).To(BeFalse())
	})

	It("should return false when neither running nor runStrategy is set", func() {
		vm := &kubevirtv1.VirtualMachine{
			Spec: kubevirtv1.VirtualMachineSpec{},
		}
		// RunStrategy() returns RunStrategyHalted by default when nothing is set
		Expect(isVMRunning(vm)).To(BeFalse())
	})
})

// =============================================================================
// Tests for runStrategyPtr
// =============================================================================

var _ = Describe("runStrategyPtr", func() {
	It("should return a pointer to RunStrategyAlways", func() {
		ptr := runStrategyPtr(kubevirtv1.RunStrategyAlways)
		Expect(ptr).ToNot(BeNil())
		Expect(*ptr).To(Equal(kubevirtv1.RunStrategyAlways))
	})

	It("should return a pointer to RunStrategyHalted", func() {
		ptr := runStrategyPtr(kubevirtv1.RunStrategyHalted)
		Expect(ptr).ToNot(BeNil())
		Expect(*ptr).To(Equal(kubevirtv1.RunStrategyHalted))
	})

	It("should return a pointer to RunStrategyOnce", func() {
		ptr := runStrategyPtr(kubevirtv1.RunStrategyOnce)
		Expect(ptr).ToNot(BeNil())
		Expect(*ptr).To(Equal(kubevirtv1.RunStrategyOnce))
	})

	It("should return a pointer to RunStrategyRerunOnFailure", func() {
		ptr := runStrategyPtr(kubevirtv1.RunStrategyRerunOnFailure)
		Expect(ptr).ToNot(BeNil())
		Expect(*ptr).To(Equal(kubevirtv1.RunStrategyRerunOnFailure))
	})
})

// =============================================================================
// Tests for quantityPtr
// =============================================================================

var _ = Describe("quantityPtr", func() {
	It("should return a pointer to 4Gi quantity", func() {
		q := resource.MustParse("4Gi")
		ptr := quantityPtr(q)
		Expect(ptr).ToNot(BeNil())
		Expect(ptr.String()).To(Equal("4Gi"))
	})

	It("should return a pointer to 100Mi quantity", func() {
		q := resource.MustParse("100Mi")
		ptr := quantityPtr(q)
		Expect(ptr).ToNot(BeNil())
		Expect(ptr.String()).To(Equal("100Mi"))
	})

	It("should return a pointer to 0 quantity", func() {
		q := resource.MustParse("0")
		ptr := quantityPtr(q)
		Expect(ptr).ToNot(BeNil())
		Expect(ptr.IsZero()).To(BeTrue())
	})

	It("should return a pointer to decimal SI CPU quantity", func() {
		q := resource.MustParse("2")
		ptr := quantityPtr(q)
		Expect(ptr).ToNot(BeNil())
		Expect(ptr.Value()).To(Equal(int64(2)))
	})
})

// =============================================================================
// Tests for buildNetworkInterfaces
// =============================================================================

var _ = Describe("buildNetworkInterfaces", func() {
	It("should create interfaces for each network", func() {
		machine := &infrav1.HarvesterMachine{
			Spec: infrav1.HarvesterMachineSpec{
				Networks: []string{"default/prod", "default/mgmt"},
			},
		}
		ifaces := buildNetworkInterfaces(machine)
		Expect(ifaces).To(HaveLen(2))
		Expect(ifaces[0].Name).To(Equal("nic-1"))
		Expect(ifaces[0].Model).To(Equal("virtio"))
		Expect(ifaces[1].Name).To(Equal("nic-2"))
		Expect(ifaces[1].Model).To(Equal("virtio"))
	})

	It("should handle single network", func() {
		machine := &infrav1.HarvesterMachine{
			Spec: infrav1.HarvesterMachineSpec{
				Networks: []string{"default/production"},
			},
		}
		ifaces := buildNetworkInterfaces(machine)
		Expect(ifaces).To(HaveLen(1))
		Expect(ifaces[0].Name).To(Equal("nic-1"))
		Expect(ifaces[0].Model).To(Equal("virtio"))
		// Check bridge binding method is set
		Expect(ifaces[0].InterfaceBindingMethod).ToNot(Equal(kubevirtv1.InterfaceBindingMethod{}))
	})

	It("should handle empty networks", func() {
		machine := &infrav1.HarvesterMachine{
			Spec: infrav1.HarvesterMachineSpec{Networks: []string{}},
		}
		ifaces := buildNetworkInterfaces(machine)
		Expect(ifaces).To(BeEmpty())
	})

	It("should handle 5 networks with correct numbering", func() {
		machine := &infrav1.HarvesterMachine{
			Spec: infrav1.HarvesterMachineSpec{
				Networks: []string{"n1", "n2", "n3", "n4", "n5"},
			},
		}
		ifaces := buildNetworkInterfaces(machine)
		Expect(ifaces).To(HaveLen(5))
		for i, iface := range ifaces {
			Expect(iface.Name).To(Equal("nic-" + strings.Replace(
				strings.Replace(string(rune('1'+i)), "\x00", "", -1), "", "", 0)))
		}
		Expect(ifaces[0].Name).To(Equal("nic-1"))
		Expect(ifaces[4].Name).To(Equal("nic-5"))
	})
})

// =============================================================================
// Tests for buildAffinity
// =============================================================================

var _ = Describe("buildAffinity", func() {
	It("should create default PodAntiAffinity", func() {
		scope := &Scope{
			HarvesterMachine: &infrav1.HarvesterMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vm"},
				Spec:       infrav1.HarvesterMachineSpec{},
			},
		}
		affinity := buildAffinity(scope)
		Expect(affinity).ToNot(BeNil())
		Expect(affinity.PodAntiAffinity).ToNot(BeNil())
		Expect(affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))
		Expect(affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Weight).To(Equal(int32(1)))
		Expect(affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.TopologyKey).To(Equal("kubernetes.io/hostname"))
		Expect(affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.LabelSelector.MatchLabels["harvesterhci.io/vmNamePrefix"]).To(Equal("test-vm"))
		Expect(affinity.NodeAffinity).To(BeNil())
		Expect(affinity.PodAffinity).To(BeNil())
	})

	It("should merge NodeAffinity when specified", func() {
		nodeAffinity := &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{MatchExpressions: []corev1.NodeSelectorRequirement{
						{Key: "kubernetes.io/hostname", Operator: corev1.NodeSelectorOpIn, Values: []string{"node1"}},
					}},
				},
			},
		}
		scope := &Scope{
			HarvesterMachine: &infrav1.HarvesterMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vm"},
				Spec: infrav1.HarvesterMachineSpec{
					NodeAffinity: nodeAffinity,
				},
			},
		}
		affinity := buildAffinity(scope)
		Expect(affinity.NodeAffinity).To(Equal(nodeAffinity))
		Expect(affinity.PodAntiAffinity).ToNot(BeNil())
		Expect(affinity.PodAffinity).To(BeNil())
	})

	It("should merge WorkloadAffinity when specified", func() {
		podAffinity := &corev1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{TopologyKey: "kubernetes.io/hostname"},
			},
		}
		scope := &Scope{
			HarvesterMachine: &infrav1.HarvesterMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vm"},
				Spec: infrav1.HarvesterMachineSpec{
					WorkloadAffinity: podAffinity,
				},
			},
		}
		affinity := buildAffinity(scope)
		Expect(affinity.PodAffinity).To(Equal(podAffinity))
		Expect(affinity.PodAntiAffinity).ToNot(BeNil())
		Expect(affinity.NodeAffinity).To(BeNil())
	})

	It("should merge both NodeAffinity and WorkloadAffinity when specified", func() {
		nodeAffinity := &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{MatchExpressions: []corev1.NodeSelectorRequirement{
						{Key: "kubernetes.io/hostname", Operator: corev1.NodeSelectorOpIn, Values: []string{"node1"}},
					}},
				},
			},
		}
		podAffinity := &corev1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{TopologyKey: "topology.kubernetes.io/zone"},
			},
		}
		scope := &Scope{
			HarvesterMachine: &infrav1.HarvesterMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "complex-vm"},
				Spec: infrav1.HarvesterMachineSpec{
					NodeAffinity:     nodeAffinity,
					WorkloadAffinity: podAffinity,
				},
			},
		}
		affinity := buildAffinity(scope)
		Expect(affinity.NodeAffinity).To(Equal(nodeAffinity))
		Expect(affinity.PodAffinity).To(Equal(podAffinity))
		Expect(affinity.PodAntiAffinity).ToNot(BeNil())
		Expect(affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.LabelSelector.MatchLabels["harvesterhci.io/vmNamePrefix"]).To(Equal("complex-vm"))
	})
})

// =============================================================================
// Tests for getCloudInitData
// =============================================================================

var _ = Describe("getCloudInitData", func() {
	It("should retrieve cloud-init data from bootstrap secret", func() {
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)

		bootstrapSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "bootstrap-data", Namespace: "test-ns"},
			Data:       map[string][]byte{"value": []byte("runcmd:\n  - echo hello\n")},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(bootstrapSecret).Build()

		dataSecretName := "bootstrap-data"
		scope := &Scope{
			Ctx: context.TODO(),
			Machine: &clusterv1.Machine{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns"},
				Spec: clusterv1.MachineSpec{
					Bootstrap: clusterv1.Bootstrap{DataSecretName: &dataSecretName},
				},
			},
			ReconcilerClient: fakeClient,
		}

		data, err := getCloudInitData(scope)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(ContainSubstring("echo hello"))
		Expect(data).To(ContainSubstring("runcmd:"))
	})

	It("should return error when secret not found", func() {
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		dataSecretName := "missing-secret"
		scope := &Scope{
			Ctx: context.TODO(),
			Machine: &clusterv1.Machine{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns"},
				Spec: clusterv1.MachineSpec{
					Bootstrap: clusterv1.Bootstrap{DataSecretName: &dataSecretName},
				},
			},
			ReconcilerClient: fakeClient,
		}

		_, err := getCloudInitData(scope)
		Expect(err).To(HaveOccurred())
	})

	It("should return error when secret has no 'value' key", func() {
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)

		bootstrapSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "no-value-secret", Namespace: "test-ns"},
			Data:       map[string][]byte{"other-key": []byte("some data")},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(bootstrapSecret).Build()

		dataSecretName := "no-value-secret"
		scope := &Scope{
			Ctx: context.TODO(),
			Machine: &clusterv1.Machine{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns"},
				Spec: clusterv1.MachineSpec{
					Bootstrap: clusterv1.Bootstrap{DataSecretName: &dataSecretName},
				},
			},
			ReconcilerClient: fakeClient,
		}

		_, err := getCloudInitData(scope)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no userData key found"))
	})

	It("should return empty string for empty 'value' key", func() {
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)

		bootstrapSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "empty-value-secret", Namespace: "test-ns"},
			Data:       map[string][]byte{"value": []byte("")},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(bootstrapSecret).Build()

		dataSecretName := "empty-value-secret"
		scope := &Scope{
			Ctx: context.TODO(),
			Machine: &clusterv1.Machine{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns"},
				Spec: clusterv1.MachineSpec{
					Bootstrap: clusterv1.Bootstrap{DataSecretName: &dataSecretName},
				},
			},
			ReconcilerClient: fakeClient,
		}

		data, err := getCloudInitData(scope)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(Equal(""))
	})
})

// =============================================================================
// Tests for dhclientScriptContent
// =============================================================================

var _ = Describe("dhclientScriptContent", func() {
	It("should return a valid bash script", func() {
		content := dhclientScriptContent()
		Expect(content).To(HavePrefix("#!/bin/bash"))
	})

	It("should handle PREINIT reason", func() {
		content := dhclientScriptContent()
		Expect(content).To(ContainSubstring("PREINIT"))
		Expect(content).To(ContainSubstring("ip link set dev"))
	})

	It("should handle BOUND/RENEW/REBIND/REBOOT reasons", func() {
		content := dhclientScriptContent()
		Expect(content).To(ContainSubstring("BOUND|RENEW|REBIND|REBOOT"))
		Expect(content).To(ContainSubstring("ip addr add"))
		Expect(content).To(ContainSubstring("ip route add default"))
	})

	It("should configure resolv.conf from DHCP", func() {
		content := dhclientScriptContent()
		Expect(content).To(ContainSubstring("/etc/resolv.conf"))
		Expect(content).To(ContainSubstring("nameserver"))
	})

	It("should convert netmask to CIDR prefix", func() {
		content := dhclientScriptContent()
		Expect(content).To(ContainSubstring("new_subnet_mask"))
		Expect(content).To(ContainSubstring("prefix"))
	})
})

// =============================================================================
// Tests for buildPVCForVolume
// =============================================================================

var _ = Describe("buildPVCForVolume", func() {
	It("should build a PVC for storageClass volume type", func() {
		size := resource.MustParse("10Gi")
		vol := &infrav1.Volume{
			VolumeType:   "storageClass",
			StorageClass: "longhorn",
			VolumeSize:   &size,
		}
		scope := &Scope{
			HarvesterMachine: &infrav1.HarvesterMachine{},
			HarvesterCluster: &infrav1.HarvesterCluster{
				Spec: infrav1.HarvesterClusterSpec{TargetNamespace: "default"},
			},
		}
		pvc, err := buildPVCForVolume(vol, "test-pvc", "default", scope)
		Expect(err).ToNot(HaveOccurred())
		Expect(pvc.Name).To(Equal("test-pvc"))
		Expect(pvc.Namespace).To(Equal("default"))
		Expect(*pvc.Spec.StorageClassName).To(Equal("longhorn"))
		Expect(pvc.Spec.AccessModes).To(ContainElement(corev1.ReadWriteMany))
		block := corev1.PersistentVolumeBlock
		Expect(*pvc.Spec.VolumeMode).To(Equal(block))
		Expect(pvc.Spec.Resources.Requests["storage"]).To(Equal(resource.MustParse("10Gi")))
	})

	It("should build a PVC for storageClass with different size", func() {
		size := resource.MustParse("100Gi")
		vol := &infrav1.Volume{
			VolumeType:   "storageClass",
			StorageClass: "harv-rep1",
			VolumeSize:   &size,
		}
		scope := &Scope{
			HarvesterMachine: &infrav1.HarvesterMachine{},
			HarvesterCluster: &infrav1.HarvesterCluster{
				Spec: infrav1.HarvesterClusterSpec{TargetNamespace: "ns1"},
			},
		}
		pvc, err := buildPVCForVolume(vol, "big-disk-0-xyz", "ns1", scope)
		Expect(err).ToNot(HaveOccurred())
		Expect(pvc.Name).To(Equal("big-disk-0-xyz"))
		Expect(pvc.Namespace).To(Equal("ns1"))
		Expect(*pvc.Spec.StorageClassName).To(Equal("harv-rep1"))
		Expect(pvc.Spec.Resources.Requests["storage"]).To(Equal(resource.MustParse("100Gi")))
	})

	It("should set correct annotations for storageClass type", func() {
		size := resource.MustParse("20Gi")
		vol := &infrav1.Volume{
			VolumeType:   "storageClass",
			StorageClass: "longhorn",
			VolumeSize:   &size,
		}
		scope := &Scope{
			HarvesterMachine: &infrav1.HarvesterMachine{},
			HarvesterCluster: &infrav1.HarvesterCluster{
				Spec: infrav1.HarvesterClusterSpec{TargetNamespace: "default"},
			},
		}
		pvc, err := buildPVCForVolume(vol, "data-pvc", "default", scope)
		Expect(err).ToNot(HaveOccurred())
		// storageClass type should NOT have imageId annotation
		_, hasImageAnnotation := pvc.Annotations[hvAnnotationImageID]
		Expect(hasImageAnnotation).To(BeFalse())
	})
})

// =============================================================================
// Tests for getKubevirtNetworksFromHarvesterMachine (additional cases)
// =============================================================================

var _ = Describe("getKubevirtNetworksFromHarvesterMachine additional", func() {
	It("should handle empty networks list", func() {
		machine := &infrav1.HarvesterMachine{
			Spec: infrav1.HarvesterMachineSpec{
				Networks: []string{},
			},
		}
		networks := getKubevirtNetworksFromHarvesterMachine(machine)
		Expect(networks).To(BeEmpty())
	})

	It("should handle single network", func() {
		machine := &infrav1.HarvesterMachine{
			Spec: infrav1.HarvesterMachineSpec{
				Networks: []string{"default/production"},
			},
		}
		networks := getKubevirtNetworksFromHarvesterMachine(machine)
		Expect(networks).To(HaveLen(1))
		Expect(networks[0].Name).To(Equal("nic-1"))
		Expect(networks[0].Multus.NetworkName).To(Equal("default/production"))
	})

	It("should handle namespaced network names", func() {
		machine := &infrav1.HarvesterMachine{
			Spec: infrav1.HarvesterMachineSpec{
				Networks: []string{"ns1/net1", "ns2/net2", "ns3/net3"},
			},
		}
		networks := getKubevirtNetworksFromHarvesterMachine(machine)
		Expect(networks).To(HaveLen(3))
		Expect(networks[0].Multus.NetworkName).To(Equal("ns1/net1"))
		Expect(networks[1].Multus.NetworkName).To(Equal("ns2/net2"))
		Expect(networks[2].Multus.NetworkName).To(Equal("ns3/net3"))
	})
})

// =============================================================================
// Tests for getWorkloadClusterConfig
// =============================================================================

var _ = Describe("getWorkloadClusterConfig", func() {
	It("should return error when kubeconfig secret does not exist", func() {
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = infrav1.AddToScheme(scheme)
		_ = clusterv1.AddToScheme(scheme)
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		logger := log.FromContext(context.TODO())
		scope := &Scope{
			Ctx:              context.TODO(),
			Logger:           &logger,
			ReconcilerClient: fakeClient,
			Cluster: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
			},
		}

		_, err := getWorkloadClusterConfig(scope)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unable to get workload cluster kubeconfig secret"))
	})

	It("should return error when secret has no value key", func() {
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = infrav1.AddToScheme(scheme)
		_ = clusterv1.AddToScheme(scheme)

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cluster-kubeconfig",
				Namespace: "test-ns",
			},
			Data: map[string][]byte{
				"other-key": []byte("not-a-kubeconfig"),
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secret).
			Build()

		logger := log.FromContext(context.TODO())
		scope := &Scope{
			Ctx:              context.TODO(),
			Logger:           &logger,
			ReconcilerClient: fakeClient,
			Cluster: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
			},
		}

		_, err := getWorkloadClusterConfig(scope)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no kubeconfig data found"))
	})

	It("should return error when kubeconfig data is invalid", func() {
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = infrav1.AddToScheme(scheme)
		_ = clusterv1.AddToScheme(scheme)

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bad-cluster-kubeconfig",
				Namespace: "test-ns",
			},
			Data: map[string][]byte{
				"value": []byte("this is not a valid kubeconfig"),
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secret).
			Build()

		logger := log.FromContext(context.TODO())
		scope := &Scope{
			Ctx:              context.TODO(),
			Logger:           &logger,
			ReconcilerClient: fakeClient,
			Cluster: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "bad-cluster", Namespace: "test-ns"},
			},
		}

		_, err := getWorkloadClusterConfig(scope)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unable to get workload cluster config"))
	})

	It("should return valid config from correct kubeconfig secret", func() {
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = infrav1.AddToScheme(scheme)
		_ = clusterv1.AddToScheme(scheme)

		kubeconfig := []byte(`apiVersion: v1
clusters:
- cluster:
    server: https://172.16.3.100:6443
  name: workload
contexts:
- context:
    cluster: workload
    user: admin
  name: workload
current-context: workload
kind: Config
users:
- name: admin
  user:
    token: test-token-123
`)

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "good-cluster-kubeconfig",
				Namespace: "test-ns",
			},
			Data: map[string][]byte{
				"value": kubeconfig,
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secret).
			Build()

		logger := log.FromContext(context.TODO())
		scope := &Scope{
			Ctx:              context.TODO(),
			Logger:           &logger,
			ReconcilerClient: fakeClient,
			Cluster: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "good-cluster", Namespace: "test-ns"},
			},
		}

		config, err := getWorkloadClusterConfig(scope)
		Expect(err).ToNot(HaveOccurred())
		Expect(config).ToNot(BeNil())
		Expect(config.Host).To(Equal("https://172.16.3.100:6443"))
	})
})

// =============================================================================
// Tests for initializeWorkloadNode (early returns)
// =============================================================================

var _ = Describe("initializeWorkloadNode", func() {
	It("should return early when ProviderID is empty", func() {
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		logger := log.FromContext(context.TODO())
		r := &HarvesterMachineReconciler{Client: fakeClient, Scheme: scheme}

		scope := &Scope{
			Ctx:    context.TODO(),
			Logger: &logger,
			HarvesterMachine: &infrav1.HarvesterMachine{
				Spec: infrav1.HarvesterMachineSpec{
					ProviderID: "", // empty -> early return
				},
			},
			Cluster: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
			},
			ReconcilerClient: fakeClient,
		}

		// Should not panic - just returns early
		r.initializeWorkloadNode(scope)
	})

	It("should return early when workload cluster config is not available", func() {
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = infrav1.AddToScheme(scheme)
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		logger := log.FromContext(context.TODO())
		r := &HarvesterMachineReconciler{Client: fakeClient, Scheme: scheme}

		scope := &Scope{
			Ctx:    context.TODO(),
			Logger: &logger,
			HarvesterMachine: &infrav1.HarvesterMachine{
				Spec: infrav1.HarvesterMachineSpec{
					ProviderID: "harvester://test-vm",
				},
			},
			Cluster: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
			},
			ReconcilerClient: fakeClient,
		}

		// Secret doesn't exist -> getWorkloadClusterConfig fails -> early return
		r.initializeWorkloadNode(scope)
	})
})

// =============================================================================
// Tests for getProviderIDFromWorkloadCluster error paths
// =============================================================================

var _ = Describe("getProviderIDFromWorkloadCluster", func() {
	It("should return error when workload cluster kubeconfig secret is missing", func() {
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = infrav1.AddToScheme(scheme)
		_ = clusterv1.AddToScheme(scheme)
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		logger := log.FromContext(context.TODO())
		scope := &Scope{
			Ctx:              context.TODO(),
			Logger:           &logger,
			ReconcilerClient: fakeClient,
			Cluster: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "no-config", Namespace: "ns"},
			},
			HarvesterMachine: &infrav1.HarvesterMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "machine-1", Namespace: "ns"},
			},
		}

		_, err := getProviderIDFromWorkloadCluster(scope)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unable to get workload cluster config"))
	})
})

// =============================================================================
// Tests for buildVMTemplate error paths (SSH key not found)
// =============================================================================

// Note: buildVMTemplate requires HarvesterClient (concrete type) for SSH key lookup.
// We test only through the dependent functions that are testable.

// =============================================================================
// Tests for buildPVCForVolume with unknown volume type
// =============================================================================

var _ = Describe("buildPVCForVolume edge cases", func() {
	It("should create PVC with no storage class for unknown volume type", func() {
		size := resource.MustParse("5Gi")
		vol := &infrav1.Volume{
			VolumeType: "unknown",
			VolumeSize: &size,
		}
		scope := &Scope{
			HarvesterMachine: &infrav1.HarvesterMachine{},
			HarvesterCluster: &infrav1.HarvesterCluster{
				Spec: infrav1.HarvesterClusterSpec{TargetNamespace: "default"},
			},
		}
		pvc, err := buildPVCForVolume(vol, "test-pvc", "default", scope)
		Expect(err).ToNot(HaveOccurred())
		Expect(pvc.Name).To(Equal("test-pvc"))
		// Unknown type should not set StorageClassName
		Expect(pvc.Spec.StorageClassName).To(BeNil())
	})

	It("should create PVC with empty annotations for storageClass type", func() {
		size := resource.MustParse("50Gi")
		vol := &infrav1.Volume{
			VolumeType:   "storageClass",
			StorageClass: "longhorn-image-xyz",
			VolumeSize:   &size,
		}
		scope := &Scope{
			HarvesterMachine: &infrav1.HarvesterMachine{},
			HarvesterCluster: &infrav1.HarvesterCluster{
				Spec: infrav1.HarvesterClusterSpec{TargetNamespace: "default"},
			},
		}
		pvc, err := buildPVCForVolume(vol, "boot-disk", "default", scope)
		Expect(err).ToNot(HaveOccurred())
		// storageClass type should NOT have imageId annotation
		_, hasImageAnnotation := pvc.Annotations[hvAnnotationImageID]
		Expect(hasImageAnnotation).To(BeFalse())
		Expect(*pvc.Spec.StorageClassName).To(Equal("longhorn-image-xyz"))
	})
})
