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
