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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	ClusterFinalizer = "harvester.infrastructure.cluster.x-k8s.io"
	DHCP             = "dhcp"
	POOL             = "pool"
)

// HarvesterClusterSpec defines the desired state of HarvesterCluster
type HarvesterClusterSpec struct {
	// Server is the url to connect to Harvester.
	// +optional
	Server string `json:"server,omitempty"`

	// IdentitySecret is the name of the Secret containing HarvesterKubeConfig file.
	IdentitySecret SecretKey `json:"identitySecret"`

	// LoadBalancerConfig describes how the load balancer should be created in Harvester.
	LoadBalancerConfig LoadBalancerConfig `json:"loadBalancerConfig"`

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

type LoadBalancerConfig struct {
	// IPAMType is the configuration of IP addressing for the control plane load balancer.
	// This can take two values, either "dhcp" or "ippool"
	IPAMType IPAMType `json:"ipamType"`

	// IpPoolRef is a reference to an existing IpPool object in Harvester's cluster in the same namespace.
	// This field is mutually exclusive with "ipPool"
	IpPoolRef string `json:"ipPoolRef,omitempty"`

	// IpPool defines a new IpPool that will be added to Harvester.
	// This field is mutually exclusive with "IpPoolRef"
	IpPool IpPool `json:"ipPool,omitempty"`
}

// IPPAMType describes the way the LoadBalancer IP should be created, using DHCP or using an IPPool defined in Harvester.
// +kubebuilder:validation:Enum:=dhcp;pool
type IPAMType string

// IpPool is a description of a new IPPool to be created in Harvester
type IpPool struct {
	// VMNetwork is the name of an existing VM Network in Harvester where the IPPool should exist.
	VMNetwork string `json:"vmNetwork"`

	// Subnet is a string describing the subnet that should be used by the IP Pool, it should have the CIDR Format of an IPv4 Address
	// e.g. 172.17.1.0/24
	Subnet string `json:"subnet"`

	// Gateway is the IP Address that should be used by the Gateway on the Subnet. It should be a valid address inside the subnet
	// e.g. 172.17.1.1
	Gateway string `json:"gateway"`
}

// HarvesterClusterStatus defines the observed state of HarvesterCluster
type HarvesterClusterStatus struct {
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

	Spec   HarvesterClusterSpec   `json:"spec"`
	Status HarvesterClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HarvesterClusterList contains a list of HarvesterCluster
type HarvesterClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HarvesterCluster `json:"items"`
}

// HarvesterClusterTemplateSpec defines the desired state of HarvesterClusterTemplate.
type HarvesterClusterTemplateSpec struct {
	Template HarvesterClusterTemplateResource `json:"template"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=infraclustertemplates,scope=Namespaced,categories=cluster-api,shortName=ict
// +kubebuilder:storageversion

// HarvesterClusterTemplate is the Schema for the infraclustertemplates API.
type HarvesterClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec HarvesterClusterTemplateSpec `json:"spec,omitempty"`
}

type HarvesterClusterTemplateResource struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty"`
	Spec       HarvesterClusterSpec `json:"spec"`
}

// HarvesterClusterTemplateList contains a list of HarvesterClusterTemplates.
type HarvesterClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HarvesterClusterTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HarvesterCluster{}, &HarvesterClusterList{})
}
