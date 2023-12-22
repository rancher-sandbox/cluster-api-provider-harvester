package util

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MergeCloudInitStrings", func() {
	var cloudinit1 string
	var cloudinit2 string
	var cloudinit3 string

	BeforeEach(func() {
		cloudinit1 = `#cloud-config
package_update: true
packages:
  - nginx
runcmd:
  - echo "hello world"
`
		cloudinit2 = `ssh_authorized_keys:
  - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAACAA... user@host`

		cloudinit3 = `#cloud-config
package_update: false
packages:
  - curl
runcmd:
  - echo "hello world 3"
`

	})
	It("Should show the right resulting cloud-init", func() {
		mergedCloudInit, err := MergeCloudInitData(cloudinit1, cloudinit2, cloudinit3)
		Expect(err).ToNot(HaveOccurred())
		mergedCloudInitString := string(mergedCloudInit)
		_, err = GinkgoWriter.Write(mergedCloudInit)
		Expect(err).NotTo(HaveOccurred())
		Expect(mergedCloudInitString).To(Equal(`#cloud-config
package_update: false
packages:
- nginx
- curl
runcmd:
- echo "hello world"
- echo "hello world 3"
ssh_authorized_keys:
- ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAACAA... user@host
`))
	})
})
