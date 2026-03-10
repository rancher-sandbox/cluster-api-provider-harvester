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
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Node initialization utilities", func() {

	Describe("hasUninitializedTaint", func() {
		It("should return true when the uninitialized taint is present", func() {
			node := &v1.Node{
				Spec: v1.NodeSpec{
					Taints: []v1.Taint{
						{Key: "node.kubernetes.io/not-ready", Effect: v1.TaintEffectNoSchedule},
						{Key: uninitializedTaintKey, Value: "true", Effect: v1.TaintEffectNoSchedule},
					},
				},
			}
			Expect(hasUninitializedTaint(node)).To(BeTrue())
		})

		It("should return false when the taint is not present", func() {
			node := &v1.Node{
				Spec: v1.NodeSpec{
					Taints: []v1.Taint{
						{Key: "node.kubernetes.io/not-ready", Effect: v1.TaintEffectNoSchedule},
					},
				},
			}
			Expect(hasUninitializedTaint(node)).To(BeFalse())
		})

		It("should return false for an empty taint list", func() {
			node := &v1.Node{
				Spec: v1.NodeSpec{Taints: nil},
			}
			Expect(hasUninitializedTaint(node)).To(BeFalse())
		})
	})

	Describe("removeTaint", func() {
		It("should remove the specified taint and keep others", func() {
			taints := []v1.Taint{
				{Key: "node.kubernetes.io/not-ready", Effect: v1.TaintEffectNoSchedule},
				{Key: uninitializedTaintKey, Value: "true", Effect: v1.TaintEffectNoSchedule},
				{Key: "other-taint", Effect: v1.TaintEffectNoExecute},
			}
			result := removeTaint(taints, uninitializedTaintKey)
			Expect(result).To(HaveLen(2))
			Expect(result[0].Key).To(Equal("node.kubernetes.io/not-ready"))
			Expect(result[1].Key).To(Equal("other-taint"))
		})

		It("should return the same slice when the key is not found", func() {
			taints := []v1.Taint{
				{Key: "node.kubernetes.io/not-ready", Effect: v1.TaintEffectNoSchedule},
			}
			result := removeTaint(taints, uninitializedTaintKey)
			Expect(result).To(HaveLen(1))
		})

		It("should handle an empty slice", func() {
			result := removeTaint([]v1.Taint{}, uninitializedTaintKey)
			Expect(result).To(BeEmpty())
		})

		It("should handle nil slice", func() {
			result := removeTaint(nil, uninitializedTaintKey)
			Expect(result).To(BeEmpty())
		})
	})
})
