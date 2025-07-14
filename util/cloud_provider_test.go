package util

import (
	"encoding/base64"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"sigs.k8s.io/yaml"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned"
)

var (
	yamlString string = `apiVersion: v1
kind: Secret
metadata:
  name: test-secret
  namespace: default
  type: Opaque
data:
  username: aGVsbG8K
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-configmap
  namespace: default
data:
  key1: value1
  key2: value2
`
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

var _ = Describe("GetConfigMapsFromYAML", func() {
	var secrets []*corev1.Secret
	indexes := make([]int, 0, maxNumberOfSecrets)

	It("Should return the right name", func() {
		objectsFromYAML, err := GetSerializedObjects(yamlString)
		Expect(err).To(BeNil())

		secrets, indexes, err = GetSecrets(objectsFromYAML)
		Expect(err).To(BeNil())

		Expect(len(secrets)).To(Equal(1))
		Expect(secrets[0].Name).To(Equal("test-secret"))
		Expect(secrets[0].Namespace).To(Equal("default"))
		Expect(len(secrets[0].Data)).To(Equal(1))
		Expect(string(secrets[0].Data["username"])).To(Equal("hello\n"))
		Expect(len(indexes)).To(Equal(1))
		Expect(indexes[0]).To(Equal(0))
	})
})

// Tests a change in a ConfigMap in YAML
var _ = Describe("ChangeValueInConfigMapInYAML", func() {
	var secretName string
	var secretNamespace string
	var key string
	var value []byte

	BeforeEach(func() {
		secretName = "test-secret"
		secretNamespace = "default"
		key = "username"
		value = []byte("new-value")
	})

	It("Should return the right name", func() {
		// Get the modified YAML string
		modifiedYAMLString, err := ModifyYAMlString(yamlString, secretName, secretNamespace, key, value)
		Expect(err).To(BeNil())
		Expect(modifiedYAMLString).To(ContainSubstring("bmV3LXZhbHVl"))
		documents := strings.Split(modifiedYAMLString, "---")
		Expect(len(documents)).To(Equal(2))
		var secret *corev1.Secret
		for _, document := range documents {
			if strings.Contains(document, "Secret") {
				err = yaml.Unmarshal([]byte(document), &secret)
				Expect(err).To(BeNil())
				Expect(secret.Data["username"]).To(Equal([]byte("new-value")))
			}
		}
	})
})

var _ = Describe("AddSecretToConfigMap", func() {
	It("Should return a YAML with an additional Secret", func() {

		// Get the modified YAML string with the new secret
		modifiedYAMLString, err := ModifyYAMlString(yamlString, "cloud-config", "test-hv", "cloud-config", []byte("new-value"))
		Expect(err).To(BeNil())
		Expect(modifiedYAMLString).To(Equal(`apiVersion: v1
data:
  username: aGVsbG8K
kind: Secret
metadata:
  name: test-secret
  namespace: default
  type: Opaque

---
apiVersion: v1
data:
  key1: value1
  key2: value2
kind: ConfigMap
metadata:
  name: test-configmap
  namespace: default

---
apiVersion: v1
data:
  cloud-config: bmV3LXZhbHVl
kind: Secret
metadata:
  creationTimestamp: null
  name: cloud-config
  namespace: test-hv
type: Opaque
`))
	})
})

type TestStruct struct {
	Spec map[string][]byte `yaml:"spec"`
}
