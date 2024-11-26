/*
Copyright 2024.

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

package controllers

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1 "github.com/rancher-sandbox/cluster-api-provider-harvester/api/v1alpha1"
	hvclient "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned"
)

var _ = Describe("Extract Server from Kubeconfig", func() {
	var kubeconfig []byte

	BeforeEach(func() {
		kubeconfig = []byte(`apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJ2akNDQVdPZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQkdNUnd3R2dZRFZRUUtFeE5rZVc1aGJXbGoKYkdsemRHVnVaWEl0YjNKbk1TWXdKQVlEVlFRRERCMWtlVzVoYldsamJHbHpkR1Z1WlhJdFkyRkFNVGN4TXpnMApNVFUwTWpBZUZ3MHlOREEwTWpNd016QTFOREphRncwek5EQTBNakV3TXpBMU5ESmFNRVl4SERBYUJnTlZCQW9UCkUyUjVibUZ0YVdOc2FYTjBaVzVsY2kxdmNtY3hKakFrQmdOVkJBTU1IV1I1Ym1GdGFXTnNhWE4wWlc1bGNpMWoKWVVBeE56RXpPRFF4TlRReU1Ga3dFd1lIS29aSXpqMENBUVlJS29aSXpqMERBUWNEUWdBRVA3V0RnRnk1NzRWVwp0SVYySzFGMExVZnE1VDJkQlFYVFovUUFIdWVqNDAzMGR1MklvN2tubzZ0SlI5OEJrNVk0bmpDK0VzT3c4UlZvCnJiWkdOVzJJdEtOQ01FQXdEZ1lEVlIwUEFRSC9CQVFEQWdLa01BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0hRWUQKVlIwT0JCWUVGTWJ1c3dyTTZEQS8vcjV2NjNhejJCU3VXSkVjTUFvR0NDcUdTTTQ5QkFNQ0Ewa0FNRVlDSVFETgpKQVhOUHFtZEY4SGViUm5IMTJkTkNVWEY0TXpTd0haSTZwZzVhNDVsd1FJaEFLNHZiRGVjTEIyVzBuQnJ1S0F2ClprNy9lb2JLT05TcEthRzBJdjhHaGhTdQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0t
    server: https://10.10.0.10/k8s/clusters/local
  name: local
contexts:
- context:
    cluster: local
    user: local
  name: deathstar
current-context: deathstar
kind: Config
preferences: {}
users:
- name: local
  user:
    token: kubeconfig-user-gqrx7b7k5x:qsp8xzzd4dpb9d99n9fg8vrtdrqndplmlfjtzpkshc9jcndn9fc2ns
`)
	})
	Context("When we provide a kubeconfig with different current-context and cluster.name", func() {
		It("Should still return the right cluster", func() {
			Expect(getHarvesterServerFromKubeconfig(kubeconfig)).To(Equal("https://10.10.0.10/k8s/clusters/local"))
		})
	})
})

var _ = Describe("Modify Cloud Provider Manifest", func() {

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/component: cloud-provider
    app.kubernetes.io/name: harvester-cloud-provider
  name: harvester-cloud-provider
  namespace: kube-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app.kubernetes.io/component: cloud-provider
      app.kubernetes.io/name: harvester-cloud-provider
  template:
    metadata:
      labels:
        app.kubernetes.io/component: cloud-provider
        app.kubernetes.io/name: harvester-cloud-provider
    spec:
      containers:
      - args:
        - --cloud-config=/etc/kubernetes/cloud-config
        command:
        - harvester-cloud-provider
        image: rancher/harvester-cloud-provider:v0.2.1
        imagePullPolicy: Always
        name: harvester-cloud-provider
        resources: {}
        volumeMounts:
        - mountPath: /etc/kubernetes
          name: cloud-config
      serviceAccountName: harvester-cloud-controller-manager
      tolerations:
      - effect: NoSchedule
        key: node-role.kubernetes.io/control-plane
        operator: Exists
      - effect: NoSchedule
        key: node.cloudprovider.kubernetes.io/uninitialized
        operator: Equal
        value: "true"
      volumes:
        - name: cloud-config
          secret:
            secretName: cloud-config
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: harvester-cloud-controller-manager
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: harvester-cloud-controller-manager
rules:
- apiGroups:
  - ""
  resources:
  - services
  - nodes
  - events
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
- apiGroups:
  - ""
  resources:
  - services/status
  verbs:
  - update
  - patch
- apiGroups:
  - ""
  resources:
  - nodes/status
  verbs:
  - patch
  - update
- apiGroups:
  - coordination.k8s.io
  resources:
  - leases
  verbs:
  - get
  - create
  - update
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: harvester-cloud-controller-manager
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: harvester-cloud-controller-manager
subjects:
  - kind: ServiceAccount
    name: harvester-cloud-controller-manager
    namespace: kube-system
---
apiVersion: v1
kind: Secret
metadata:
  name: cloud-config
  namespace: kube-system
type: Opaque
data:
  cloud-config: YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOgotIG5hbWU6ICJsb2NhbCIKICBjbHVzdGVyOgogICAgc2VydmVyOiAiaHR0cHM6Ly8xOTIuMTY4LjEuMTA5L2s4cy9jbHVzdGVycy9sb2NhbCIKICAgIGNlcnRpZmljYXRlLWF1dGhvcml0eS1kYXRhOiAiTFMwdExTMUNSVWRKVGlCRFJWSlVTVVpKUTBGVVJTMHRMUzB0Q2sxSlNVSjJha05EUVwKICAgICAgVmRQWjBGM1NVSkJaMGxDUVVSQlMwSm5aM0ZvYTJwUFVGRlJSRUZxUWtkTlVuZDNSMmRaUkZaUlVVdEZlRTVyWlZjMWFHSlhiR29LWVwKICAgICAga2RzZW1SSFZuVmFXRWwwWWpOS2JrMVRXWGRLUVZsRVZsRlJSRVJDTVd0bFZ6Vm9ZbGRzYW1KSGJIcGtSMVoxV2xoSmRGa3lSa0ZOVlwKICAgICAgRmt6VGtSQk1BcE5WRkY1VGxSQlpVWjNNSGxOZWtGNFRWUm5lRTFVVFhkTmFsWmhSbmN3ZWsxNlFYaE5WRlY0VFZSTmQwMXFWbUZOUlwKICAgICAgVmw0U0VSQllVSm5UbFpDUVc5VUNrVXlValZpYlVaMFlWZE9jMkZZVGpCYVZ6VnNZMmt4ZG1OdFkzaEtha0ZyUW1kT1ZrSkJUVTFJVlwKICAgICAgMUkxWW0xR2RHRlhUbk5oV0U0d1dsYzFiR05wTVdvS1dWVkJlRTVxWXpCTlJGRjRUa1JKTVUxR2EzZEZkMWxJUzI5YVNYcHFNRU5CVVwKICAgICAgVmxKUzI5YVNYcHFNRVJCVVdORVVXZEJSVTVGVjJSU1lXTkVWSG80Y2dwdWRXaE9lV2d3YW5od1QxVlJUVGwwUmt0eFkwdDZjVEl3UVwKICAgICAgVzlPVEdNNVdsazFNMk5vVWxaV1V6QnBWamhwYW1wUk0yTTBjMHBRV0dwV1lYVlJNRVJTQ2k5TFRXNTBTVUl6VTJGT1EwMUZRWGRFWlwKICAgICAgMWxFVmxJd1VFRlJTQzlDUVZGRVFXZExhMDFCT0VkQk1WVmtSWGRGUWk5M1VVWk5RVTFDUVdZNGQwaFJXVVFLVmxJd1QwSkNXVVZHUVwKICAgICAgU3R0U0hOTGFWTXJiMHBSVGtKUlpXNUdRa2xQY2xnd1prTkJUVUZ2UjBORGNVZFRUVFE1UWtGTlEwRXdhMEZOUlZsRFNWRkRaQXBZUVwKICAgICAgV3hCUldsaE1ISnNNek5hYVhWd1ZtTjRZVTAwV0RWYU1FWXJRWEV5UlRWaE5WVmlSMHN3WW5kSmFFRk9TVlEzYzNwUVJ6Y3hSM0JNU1wKICAgICAgVXd3ZVdSakNrWnhhSEZIVWtWSlZuZzViVmRCWW04M1VEUjRTa2hqTXdvdExTMHRMVVZPUkNCRFJWSlVTVVpKUTBGVVJTMHRMUzB0IgoKdXNlcnM6Ci0gbmFtZTogImxvY2FsIgogIHVzZXI6CiAgICB0b2tlbjogImt1YmVjb25maWctdXNlci02bWx3cHY0ano1OmN2cjVrYnB3cGN6dGNwZnE0ZnZiamdkbTd4Z3BqcmtuNnBoOG1oYmZzeHpuOTJnZDdmNHo2cSIKCgpjb250ZXh0czoKLSBuYW1lOiAibG9jYWwiCiAgY29udGV4dDoKICAgIHVzZXI6ICJsb2NhbCIKICAgIGNsdXN0ZXI6ICJsb2NhbCIKICAgIG5hbWVzcGFjZTogImRlZmF1bHQiCgpjdXJyZW50LWNvbnRleHQ6ICJsb2NhbCIK
`
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "harvester-csi-driver-addon",
				Namespace: "test-hv",
			},
			Data: map[string]string{"harvester-cloud-provider-deploy.yaml": manifest},
		}).Build()
	var r *HarvesterClusterReconciler
	var scope *ClusterScope
	var log logr.Logger = log.FromContext(context.TODO())
	BeforeEach(func() {
		hvConfig, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
		Expect(err).To(BeNil())
		hvClient, err := hvclient.NewForConfig(hvConfig)
		Expect(err).To(BeNil())

		r = &HarvesterClusterReconciler{
			Client: fakeClient,
			Scheme: scheme,
		}

		scope = &ClusterScope{
			Ctx:    context.TODO(),
			Logger: log,
			Cluster: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-2",
					Namespace: "test-hv",
				},
				Spec: clusterv1.ClusterSpec{},
			},
			HarvesterCluster: &infrav1.HarvesterCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-2-hv",
					Namespace: "test-hv",
				},
				Spec: infrav1.HarvesterClusterSpec{
					TargetNamespace: "default",
					Server:          "https://192.168.1.109:6443",
					UpdateCloudProviderConfig: infrav1.UpdateCloudProviderConfig{
						ManifestsConfigMapNamespace:      "test-hv",
						ManifestsConfigMapName:           "harvester-csi-driver-addon",
						ManifestsConfigMapKey:            "harvester-cloud-provider-deploy.yaml",
						CloudConfigCredentialsSecretName: "cloud-config",
						CloudConfigCredentialsSecretKey:  "cloud-config",
					},
				},
			},
			HarvesterClient: hvClient,
		}

	})

	It("Should modify the cloud provider manifest", func() {

		Expect(r.reconcileCloudProviderConfig(scope)).To(Succeed())
		newCM := &corev1.ConfigMap{}
		Expect(fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: "test-hv", Name: "harvester-csi-driver-addon"}, newCM)).To(Succeed())
		Expect(newCM.Data["harvester-cloud-provider-deploy.yaml"]).To(Not(Equal(manifest)))

	})

})
