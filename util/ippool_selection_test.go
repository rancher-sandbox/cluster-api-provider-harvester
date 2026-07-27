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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lbv1beta1 "github.com/harvester/harvester-load-balancer/pkg/apis/loadbalancer.harvesterhci.io/v1beta1"
)

func poolWithNetwork(network string) *lbv1beta1.IPPool {
	return &lbv1beta1.IPPool{
		Spec: lbv1beta1.IPPoolSpec{
			Selector: lbv1beta1.Selector{Network: network},
		},
	}
}

var _ = Describe("PoolMatchesNetworks", func() {
	It("matches any machine when the pool has no selector network", func() {
		Expect(PoolMatchesNetworks(poolWithNetwork(""), []string{"default/production"})).To(BeTrue())
		Expect(PoolMatchesNetworks(poolWithNetwork(""), nil)).To(BeTrue())
	})

	It("matches when the selector network equals a machine network", func() {
		Expect(PoolMatchesNetworks(poolWithNetwork("default/net-cp"),
			[]string{"default/net-cp"})).To(BeTrue())
		Expect(PoolMatchesNetworks(poolWithNetwork("default/net-cp"),
			[]string{"default/net-workers", "default/net-cp"})).To(BeTrue())
	})

	It("rejects a pool assigned to another network", func() {
		Expect(PoolMatchesNetworks(poolWithNetwork("default/net-cp"),
			[]string{"default/net-workers"})).To(BeFalse())
	})

	It("compares unqualified names against the name part", func() {
		Expect(PoolMatchesNetworks(poolWithNetwork("net-cp"),
			[]string{"default/net-cp"})).To(BeTrue())
		Expect(PoolMatchesNetworks(poolWithNetwork("default/net-cp"),
			[]string{"net-cp"})).To(BeTrue())
	})

	It("does not cross namespaces when both sides are qualified", func() {
		Expect(PoolMatchesNetworks(poolWithNetwork("ns1/net"),
			[]string{"ns2/net"})).To(BeFalse())
	})
})
