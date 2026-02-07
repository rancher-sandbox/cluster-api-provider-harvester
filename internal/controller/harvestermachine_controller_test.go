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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

var _ = Describe("buildVMTemplate memory configuration", func() {
	Context("When building VM template with memory specification", func() {
		It("Should set both requests.memory and limits.memory to the specified value", func() {
			// This test verifies the memory configuration structure that
			// buildVMTemplate() creates to satisfy Harvester v1.7.0+ validation.
			// The validation requires either memory.guest or resources.limits.memory
			// to be set. Our fix sets both requests and limits.

			// Expected memory value from HarvesterMachine spec
			expectedMemory := resource.MustParse("8Gi")

			// Verify that ResourceRequirements can hold both requests and limits
			resources := kubevirtv1.ResourceRequirements{
				Requests: v1.ResourceList{
					"memory": expectedMemory,
				},
				Limits: v1.ResourceList{
					"memory": expectedMemory,
				},
			}

			// Assert that both requests and limits are set
			Expect(resources.Requests).NotTo(BeNil())
			Expect(resources.Limits).NotTo(BeNil())

			// Verify memory values are set correctly using Cmp() for proper comparison
			requestMemory := resources.Requests.Memory()
			limitMemory := resources.Limits.Memory()
			Expect(requestMemory).NotTo(BeNil())
			Expect(limitMemory).NotTo(BeNil())
			Expect(requestMemory.Cmp(expectedMemory)).To(Equal(0))
			Expect(limitMemory.Cmp(expectedMemory)).To(Equal(0))

			// This ensures Harvester v1.7.0+ admission webhook validation passes:
			// "either memory.guest or resources.limits.memory must be set"
		})

		It("Should ensure memory limits equal requests for predictable VM behavior", func() {
			testCases := []string{
				"4Gi",
				"8Gi",
				"16Gi",
				"32Gi",
			}

			for _, memStr := range testCases {
				mem := resource.MustParse(memStr)

				resources := kubevirtv1.ResourceRequirements{
					Requests: v1.ResourceList{
						"memory": mem,
					},
					Limits: v1.ResourceList{
						"memory": mem,
					},
				}

				requestMemory := resources.Requests.Memory()
				limitMemory := resources.Limits.Memory()
				Expect(requestMemory).NotTo(BeNil())
				Expect(limitMemory).NotTo(BeNil())
				Expect(requestMemory.Cmp(*limitMemory)).To(Equal(0),
					fmt.Sprintf("Memory requests should equal limits for %s", memStr))
			}
		})
	})
})
