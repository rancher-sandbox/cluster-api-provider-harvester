package util

import (
	"encoding/base64"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var _ = Describe("GetKubeconfigFromClusterAndCheck", func() {
	var hvKubeconfigB64 string
	var resultingKubeconfigB64 string
	var err error
	var saName string
	var harvesterServerURL string
	var kubeconfigBytes []byte
	var hvRESTConfig *rest.Config
	var hvClient *versioned.Clientset

	BeforeEach(func() {
		namespace = "default"
		saName = "test-1"
		hvKubeconfigB64 = os.Getenv("HV_KUBECONFIG_B64")
		harvesterServerURL = os.Getenv("HV_SERVER_URL")
	})

	It("Should return the right name", func() {
		// Build a clientset from the kubeconfig
		kubeconfigBytes, err = base64.StdEncoding.DecodeString(hvKubeconfigB64)
		Expect(err).To(BeNil())

		hvRESTConfig, err = clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
		Expect(err).To(BeNil())

		hvClient, err = versioned.NewForConfig(hvRESTConfig)
		Expect(err).To(BeNil())

		// Use the GetCloudConfigB64 function and get the resulting cloud-config B64 encoded string
		resultingKubeconfigB64, err = GetCloudConfigB64(hvClient, saName, namespace, harvesterServerURL)
		Expect(err).To(BeNil())

		// Decode the resulting cloud-config B64 encoded string and validate it
		err = ValidateB64Kubeconfig(resultingKubeconfigB64)
		Expect(err).To(BeNil())
	})
})
