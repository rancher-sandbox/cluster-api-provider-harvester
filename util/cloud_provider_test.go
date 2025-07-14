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

// Package util provides utility functions for the project.
package util

import (
	"encoding/base64"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/yaml"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned"
)

var yamlString = `apiVersion: v1
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
		Expect(err).ToNot(HaveOccurred())

		hvRESTConfig, err = clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
		Expect(err).ToNot(HaveOccurred())

		hvClient, err = versioned.NewForConfig(hvRESTConfig)
		Expect(err).ToNot(HaveOccurred())

		// Use the GetCloudConfigB64 function and get the resulting cloud-config B64 encoded string
		resultingKubeconfigB64, err = GetCloudConfigB64(hvClient, saName, namespace, harvesterServerURL)
		Expect(err).ToNot(HaveOccurred())

		// Decode the resulting cloud-config B64 encoded string and validate it
		err = ValidateB64Kubeconfig(resultingKubeconfigB64)
		Expect(err).ToNot(HaveOccurred())
	})
})

var _ = Describe("GetConfigMapsFromYAML", func() {
	var secrets []*corev1.Secret
	indexes := make([]int, 0, maxNumberOfSecrets)

	It("Should return the right name", func() {
		objectsFromYAML, err := GetSerializedObjects(yamlString)
		Expect(err).ToNot(HaveOccurred())

		secrets, indexes, err = GetSecrets(objectsFromYAML)
		Expect(err).ToNot(HaveOccurred())

		Expect(secrets).To(HaveLen(1))
		Expect(secrets[0].Name).To(Equal("test-secret"))
		Expect(secrets[0].Namespace).To(Equal("default"))
		Expect(secrets[0].Data).To(HaveLen(1))
		Expect(string(secrets[0].Data["username"])).To(Equal("hello\n"))
		Expect(indexes).To(HaveLen(1))
		Expect(indexes[0]).To(Equal(0))
	})
})

// Tests a change in a ConfigMap in YAML.
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
		Expect(err).ToNot(HaveOccurred())
		Expect(modifiedYAMLString).To(ContainSubstring("bmV3LXZhbHVl"))
		documents := strings.Split(modifiedYAMLString, "---")
		Expect(documents).To(HaveLen(2))
		var secret *corev1.Secret
		for _, document := range documents {
			if strings.Contains(document, "Secret") {
				err = yaml.Unmarshal([]byte(document), &secret)
				Expect(err).ToNot(HaveOccurred())
				Expect(secret.Data["username"]).To(Equal([]byte("new-value")))
			}
		}
	})
})

var _ = Describe("AddSecretToConfigMap", func() {
	It("Should return a YAML with an additional Secret", func() {
		// Get the modified YAML string with the new secret
		modifiedYAMLString, err := ModifyYAMlString(yamlString, "cloud-config", "test-hv", "cloud-config", []byte("new-value"))
		Expect(err).ToNot(HaveOccurred())
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
