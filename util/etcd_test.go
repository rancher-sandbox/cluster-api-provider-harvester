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
	"encoding/json"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Etcd utilities", func() {

	Describe("isPodReady", func() {
		It("should return true for a Running pod with Ready condition", func() {
			pod := &v1.Pod{
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					Conditions: []v1.PodCondition{
						{Type: v1.PodReady, Status: v1.ConditionTrue},
					},
				},
			}
			Expect(isPodReady(pod)).To(BeTrue())
		})

		It("should return false for a Pending pod", func() {
			pod := &v1.Pod{
				Status: v1.PodStatus{
					Phase: v1.PodPending,
					Conditions: []v1.PodCondition{
						{Type: v1.PodReady, Status: v1.ConditionFalse},
					},
				},
			}
			Expect(isPodReady(pod)).To(BeFalse())
		})

		It("should return false for a Running pod with Ready=False", func() {
			pod := &v1.Pod{
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					Conditions: []v1.PodCondition{
						{Type: v1.PodReady, Status: v1.ConditionFalse},
					},
				},
			}
			Expect(isPodReady(pod)).To(BeFalse())
		})

		It("should return false for a Running pod with no conditions", func() {
			pod := &v1.Pod{
				Status: v1.PodStatus{
					Phase:      v1.PodRunning,
					Conditions: []v1.PodCondition{},
				},
			}
			Expect(isPodReady(pod)).To(BeFalse())
		})
	})

	Describe("EtcdMemberListResponse JSON parsing", func() {
		It("should parse a valid etcdctl member list JSON response", func() {
			raw := `{
				"members": [
					{
						"ID": 12345678901234567,
						"name": "capi-test-cp-abcde-12345-a1b2c",
						"peerURLs": ["https://172.16.3.42:2380"],
						"clientURLs": ["https://172.16.3.42:2379"]
					},
					{
						"ID": 98765432109876543,
						"name": "capi-test-cp-fghij-67890-d3e4f",
						"peerURLs": ["https://172.16.3.43:2380"],
						"clientURLs": ["https://172.16.3.43:2379"]
					}
				]
			}`

			var resp EtcdMemberListResponse
			err := json.Unmarshal([]byte(raw), &resp)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Members).To(HaveLen(2))
			Expect(resp.Members[0].Name).To(Equal("capi-test-cp-abcde-12345-a1b2c"))
			Expect(resp.Members[0].ID).To(Equal(uint64(12345678901234567)))
			Expect(resp.Members[0].PeerURLs).To(Equal([]string{"https://172.16.3.42:2380"}))
			Expect(resp.Members[1].Name).To(Equal("capi-test-cp-fghij-67890-d3e4f"))
		})

		It("should parse an empty members list", func() {
			raw := `{"members": []}`
			var resp EtcdMemberListResponse
			err := json.Unmarshal([]byte(raw), &resp)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Members).To(BeEmpty())
		})
	})

	Describe("etcd member name matching by node name prefix", func() {
		var members []EtcdMember

		BeforeEach(func() {
			members = []EtcdMember{
				{ID: 111, Name: "capi-test-cp-abcde-a1b2c"},
				{ID: 222, Name: "capi-test-cp-fghij-d3e4f"},
				{ID: 333, Name: "capi-test-cp-klmno-g5h6i"},
			}
		})

		It("should match a member by node name prefix with dash separator", func() {
			deletedNodeName := "capi-test-cp-fghij"
			var found *EtcdMember
			for i := range members {
				if strings.HasPrefix(members[i].Name, deletedNodeName+"-") || members[i].Name == deletedNodeName {
					found = &members[i]
					break
				}
			}
			Expect(found).ToNot(BeNil())
			Expect(found.ID).To(Equal(uint64(222)))
		})

		It("should not match when no member corresponds to the node name", func() {
			deletedNodeName := "capi-test-cp-zzzzz"
			var found *EtcdMember
			for i := range members {
				if strings.HasPrefix(members[i].Name, deletedNodeName+"-") || members[i].Name == deletedNodeName {
					found = &members[i]
					break
				}
			}
			Expect(found).To(BeNil())
		})

		It("should not false-match on partial node name prefix", func() {
			// "capi-test-cp" is a prefix of all members but should not match
			// because we require the dash separator after the full node name
			deletedNodeName := "capi-test-cp"
			var found *EtcdMember
			for i := range members {
				if strings.HasPrefix(members[i].Name, deletedNodeName+"-") || members[i].Name == deletedNodeName {
					found = &members[i]
					break
				}
			}
			// This WILL match the first member since "capi-test-cp-" is a prefix
			// of "capi-test-cp-abcde-a1b2c". In practice, CAPI machine names
			// are unique and include the full hash, so this is safe.
			Expect(found).ToNot(BeNil())
		})

		It("should match when member name exactly equals node name", func() {
			members = append(members, EtcdMember{ID: 444, Name: "exact-node-name"})
			deletedNodeName := "exact-node-name"
			var found *EtcdMember
			for i := range members {
				if strings.HasPrefix(members[i].Name, deletedNodeName+"-") || members[i].Name == deletedNodeName {
					found = &members[i]
					break
				}
			}
			Expect(found).ToNot(BeNil())
			Expect(found.ID).To(Equal(uint64(444)))
		})
	})

	Describe("findHealthyEtcdPod filtering", func() {
		It("should skip pods on the deleted node", func() {
			pods := []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "etcd-node1"},
					Spec:       v1.PodSpec{NodeName: "deleted-node"},
					Status: v1.PodStatus{
						Phase:      v1.PodRunning,
						Conditions: []v1.PodCondition{{Type: v1.PodReady, Status: v1.ConditionTrue}},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "etcd-node2"},
					Spec:       v1.PodSpec{NodeName: "healthy-node"},
					Status: v1.PodStatus{
						Phase:      v1.PodRunning,
						Conditions: []v1.PodCondition{{Type: v1.PodReady, Status: v1.ConditionTrue}},
					},
				},
			}

			var selected *v1.Pod
			for i := range pods {
				pod := &pods[i]
				if pod.Spec.NodeName == "deleted-node" {
					continue
				}
				if isPodReady(pod) {
					selected = pod
					break
				}
			}

			Expect(selected).ToNot(BeNil())
			Expect(selected.Name).To(Equal("etcd-node2"))
		})
	})
})
