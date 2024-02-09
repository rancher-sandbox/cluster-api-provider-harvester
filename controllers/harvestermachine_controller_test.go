package controllers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/cluster-api-provider-harvester/api/v1alpha1"
	kubevirtv1 "kubevirt.io/api/core/v1"
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
