/*
Copyright 2022.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

const (
	ClusterFinalizer = "harvester.infrastructure.cluster.x-k8s.io"
)

// HarvesterClusterSpec defines the desired state of HarvesterCluster
type HarvesterClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Server is the url to connect to Harvester
	// +optional
	Server string `json:"server,omitempty"`

	// IdentitySecret is the name of the Secret containing HarvesterKubeConfig file
	IdentitySecret SecretKey `json:"identitySecret"`

	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint,omitempty"`
}

type SecretKey struct {

	// Namespace is the namespace in which the required Identity Secret should be found
	Namespace string `json:"namespace"`

	// Name is the name of the required Identity Secret
	Name string `json:"name"`
}

// HarvesterClusterStatus defines the observed state of HarvesterCluster
type HarvesterClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Reddy describes if the Harvester Cluster can be considered ready for machine creation
	Ready bool `json:"ready"`

	// FailureReason is the short name for the reason why a failure might be happening that makes the cluster not ready
	// +optional
	FailureReason string `json:"failureReason,omitempty"`
	// FailureMessage is a full error message dump of the above failureReason
	// +optional
	FailureMessage string `json:"failureMessage,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Cluster infrastructure is ready for HarvesterMachine"
// +kubebuilder:printcolumn:name="Server",type="string",JSONPath=".spec.server",description="Server is the address of the Harvester endpoint"
// +kubebuilder:printcolumn:name="ControlPlaneEndpoint",type="string",JSONPath=".spec.controlPlaneEndpoint[0]",description="API Endpoint",priority=1

// HarvesterCluster is the Schema for the harvesterclusters API
type HarvesterCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HarvesterClusterSpec   `json:"spec,omitempty"`
	Status HarvesterClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HarvesterClusterList contains a list of HarvesterCluster
type HarvesterClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HarvesterCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HarvesterCluster{}, &HarvesterClusterList{})
}
