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
)

var (
	namespace             string
	name                  string
	expectedResultingName string
)

var _ = Describe("GenerateNameFromFirstComponentWithDigit", func() {
	BeforeEach(func() {
		namespace = "1test"
		name = "test-1"

		expectedResultingName = "a-1test-test-1"
	})

	It("Should return the right name", func() {
		name := GenerateRFC1035Name([]string{namespace, name})
		Expect(name).To(Equal(expectedResultingName))
	})
})

var _ = Describe("GenerateNameFromUpperCase", func() {
	BeforeEach(func() {
		namespace = "TeStNS"
		name = "test"

		expectedResultingName = "testns-test"
	})

	It("Should return the right name", func() {
		name := GenerateRFC1035Name([]string{namespace, name})
		Expect(name).To(Equal(expectedResultingName))
	})
})

var _ = Describe("GenerateNameLimitedTo63", func() {
	BeforeEach(func() {
		namespace = "1TeStNS123"
		name = "namewith20characters-namewith20characters-namewith20characters"

		expectedResultingName = "a-1testns123-namewith20characters-namewith20characters-namewith"
	})

	It("Should return the right name", func() {
		name := GenerateRFC1035Name([]string{namespace, name})
		Expect(name).To(Equal(expectedResultingName))
	})
})

var _ = Describe("CheckNamespacedName", func() {
	It("should accept valid namespace/name format", func() {
		Expect(CheckNamespacedName("default/production")).To(BeTrue())
	})

	It("should accept names with dots and underscores", func() {
		Expect(CheckNamespacedName("default/sles15-sp7-minimal-vm.x86_64-cloud-qu2.qcow2")).To(BeTrue())
	})

	It("should reject names without a slash", func() {
		Expect(CheckNamespacedName("production")).To(BeFalse())
	})

	It("should reject empty string", func() {
		Expect(CheckNamespacedName("")).To(BeFalse())
	})

	It("should reject names with uppercase", func() {
		Expect(CheckNamespacedName("Default/Production")).To(BeFalse())
	})

	It("should reject names with multiple slashes", func() {
		Expect(CheckNamespacedName("a/b/c")).To(BeFalse())
	})

	It("should reject names with spaces", func() {
		Expect(CheckNamespacedName("default/my name")).To(BeFalse())
	})
})

var _ = Describe("GetNamespacedName", func() {
	It("should split namespace/name correctly", func() {
		nn, err := GetNamespacedName("default/production", "fallback-ns")
		Expect(err).ToNot(HaveOccurred())
		Expect(nn.Namespace).To(Equal("default"))
		Expect(nn.Name).To(Equal("production"))
	})

	It("should use alternative namespace when only name is given", func() {
		nn, err := GetNamespacedName("production", "fallback-ns")
		Expect(err).ToNot(HaveOccurred())
		Expect(nn.Namespace).To(Equal("fallback-ns"))
		Expect(nn.Name).To(Equal("production"))
	})

	It("should return error for malformed reference", func() {
		_, err := GetNamespacedName("Invalid Name!", "fallback-ns")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("malformed reference"))
	})

	It("should handle names with dots and underscores in namespace/name format", func() {
		nn, err := GetNamespacedName("default/image.x86_64", "fallback-ns")
		Expect(err).ToNot(HaveOccurred())
		Expect(nn.Namespace).To(Equal("default"))
		Expect(nn.Name).To(Equal("image.x86_64"))
	})

	It("should handle names with dots when using fallback namespace", func() {
		nn, err := GetNamespacedName("my-resource.name", "fallback-ns")
		Expect(err).ToNot(HaveOccurred())
		Expect(nn.Namespace).To(Equal("fallback-ns"))
		Expect(nn.Name).To(Equal("my-resource.name"))
	})
})
