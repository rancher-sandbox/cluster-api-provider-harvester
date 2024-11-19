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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	ClusterFinalizer = "harvester.infrastructure.cluster.x-k8s.io"
	DHCP             = "dhcp"
	POOL             = "pool"
)

const (
	// LoadBalancerReadyCondition documents the status of the load balancer in Harvester.
	LoadBalancerReadyCondition clusterv1.ConditionType = "LoadBalancerReady"
	// LoadBalancerNotReadyReason documents the reason why the load balancer is not ready.
	LoadBalancerNotReadyReason = "LoadBalancerNotReady"
	// LoadBalancerReadyMessage documents the message why the load balancer is not ready.
	LoadBalancerNoBackendMachineReason = "There are no machines matching the load balancer configuration"
	// LoadBalancerHealthcheckFailedReason documents the reason why the load balancer is not ready.
	LoadBalancerHealthcheckFailedReason = "The healthcheck for the load balancer failed"
	// CustomIPPoolCreatedCondition documents if a custom IP Pool was created in Harvester.
	CustomIPPoolCreatedCondition clusterv1.ConditionType = "CustomIPPoolCreated"
	// CustomPoolCreationInHarvesterFailedReason documents the reason why a custom pool was unable to be created.
	CustomPoolCreationInHarvesterFailedReason = "The custom Pool creation in Harvester failed"
	// CustomIPPoolCreatedSuccessfullyReason documents the reason why Custom IP Pool was created.
	CustomIPPoolCreatedSuccessfullyReason = "Custom IP Pool was successfully created"
)

const (
	// InitMachineCreatedCondition documents the status of the init machine in Harvester.
	InitMachineCreatedCondition clusterv1.ConditionType = "InitMachineCreated"
	// InitMachineNotCreatedReason documents the reason why the init machine is not ready.
	InitMachineNotYetCreatedReason = "InitMachineNotYetCreated"
)

// HarvesterClusterSpec defines the desired state of HarvesterCluster.
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

	// TargetNamespace is the namespace on the Harvester cluster where VMs, Load Balancers, etc. should be created.
	TargetNamespace string `json:"targetNamespace"`
}

type SecretKey struct {
	// Namespace is the namespace in which the required Identity Secret should be found.
	Namespace string `json:"namespace"`

	// Name is the name of the required Identity Secret.
	Name string `json:"name"`
}

type LoadBalancerConfig struct {
	// IPAMType is the configuration of IP addressing for the control plane load balancer.
	// This can take two values, either "dhcp" or "ippool".
	IPAMType IPAMType `json:"ipamType"`

	// IpPoolRef is a reference to an existing IpPool object in Harvester's cluster.
	// This field is mutually exclusive with "ipPool".
	IpPoolRef string `json:"ipPoolRef,omitempty"`

	// IpPool defines a new IpPool that will be added to Harvester.
	// This field is mutually exclusive with "IpPoolRef".
	IpPool IpPool `json:"ipPool,omitempty"`

	// Listeners is a list of listeners that should be created on the load balancer.
	// +optional
	Listeners []Listener `json:"listeners,omitempty"`

	// Description is a description of the load balancer that should be created.
	// +optional
	Description string `json:"description,omitempty"`
}

// IPPAMType describes the way the LoadBalancer IP should be created, using DHCP or using an IPPool defined in Harvester.
// +kubebuilder:validation:Enum:=dhcp;pool
type IPAMType string

// IpPool is a description of a new IPPool to be created in Harvester.
type IpPool struct {
	// VMNetwork is the name of an existing VM Network in Harvester where the IPPool should exist.
	// The reference can have the format "namespace/name" or just "name" if the object is in the same namespace as the HarvesterCluster.
	VMNetwork string `json:"vmNetwork"`

	// Subnet is a string describing the subnet that should be used by the IP Pool, it should have the CIDR Format of an IPv4 Address.
	// e.g. 172.17.1.0/24.
	Subnet string `json:"subnet"`

	// Gateway is the IP Address that should be used by the Gateway on the Subnet. It should be a valid address inside the subnet.
	// e.g. 172.17.1.1.
	Gateway string `json:"gateway"`

	// RangeStart is the first IP Address that should be used by the IP Pool.
	// + optional
	RangeStart string `json:"rangeStart,omitempty"`

	// RangeEnd is the last IP Address that should be used by the IP Pool.
	// + optional
	RangeEnd string `json:"rangeEnd,omitempty"`
}

// Listener is a description of a new Listener to be created on the Load Balancer.
type Listener struct {
	// Name is the name of the listener.
	Name string `json:"name"`

	// Port is the port that the listener should listen on.
	Port int32 `json:"port"`

	// Protocol is the protocol that the listener should use, either TCP or UDP.
	// +kubebuilder:validation:Enum:=TCP;UDP
	Protocol corev1.Protocol `json:"protocol"`

	// TargetPort is the port that the listener should forward traffic to.
	BackendPort int32 `json:"backendPort"`
}

// HarvesterClusterStatus defines the observed state of HarvesterCluster.
type HarvesterClusterStatus struct {
	// Ready describes if the Harvester Cluster can be considered ready for machine creation.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// FailureReason is the short name for the reason why a failure might be happening that makes the cluster not ready.
	// +optional
	FailureReason string `json:"failureReason,omitempty"`
	// FailureMessage is a full error message dump of the above failureReason.
	// +optional
	FailureMessage string `json:"failureMessage,omitempty"`

	// Conditions defines current service state of the Harvester cluster.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Cluster infrastructure is ready for HarvesterMachine"
// +kubebuilder:printcolumn:name="Server",type="string",JSONPath=".spec.server",description="Server is the address of the Harvester endpoint"
// +kubebuilder:printcolumn:name="ControlPlaneEndpoint",type="string",JSONPath=".spec.controlPlaneEndpoint[0]",description="API Endpoint",priority=1

// HarvesterCluster is the Schema for the harvesterclusters API.
type HarvesterCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HarvesterClusterSpec   `json:"spec"`
	Status HarvesterClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HarvesterClusterList contains a list of HarvesterCluster.
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
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata.
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

// GetConditions returns the set of conditions for this object.
func (m *HarvesterCluster) GetConditions() clusterv1.Conditions {
	return m.Status.Conditions
}

// SetConditions sets the conditions on this object.
func (m *HarvesterCluster) SetConditions(conditions clusterv1.Conditions) {
	m.Status.Conditions = conditions
}
