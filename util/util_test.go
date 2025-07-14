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
