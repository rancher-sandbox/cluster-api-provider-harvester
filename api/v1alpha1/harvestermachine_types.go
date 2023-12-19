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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// MachineFinalizer allows ReconcileHarvesterMachine to clean up resources associated with HarvesterMachine before
	// removing it from the apiserver.
	MachineFinalizer = "harvestermachine.infrastructure.cluster.x-k8s.io"
)

// HarvesterMachineSpec defines the desired state of HarvesterMachine
type HarvesterMachineSpec struct {
	// ProviderID will be the ID of the machine used by the controller.
	// This will be "<harvester vm namespace>-<harvester vm name>"
	// +optional
	ProviderID string `json:"providerID,omitempty"`

	// FailureDomain defines the zone or failure domain where this VM should be.
	// +optional
	FailureDomain string `json:"failureDomain,omitempty"`

	// CPU is the number of CPU to assign to the VM.
	CPU int `json:"cpu"`

	// Memory is the memory size to assign to the VM (should be similar to pod.spec.containers.resources.limits)
	Memory string `json:"memory"`

	// SSHUser is the user that should be used to connect to the VMs using SSH.
	SSHUser string `json:"sshUser"`

	//SSHKeyPair is the name of the SSH key pair to use for SSH access to the VM (this keyPair should be created in Harvester)
	SSHKeyPair string `json:"sshKeyPair"`

	// Volumes is a list of Volumes to attach to the VM
	Volumes []Volume `json:"volumes"`

	// Networks is a list of Networks to attach to the VM. Networks are referenced by their names.
	Networks []string `json:"networks"`

	// NodeAffinity gives the possibility to select preferred nodes for VM scheduling on Harvester. This works exactly like Pods.
	// +optional
	NodeAffinity *corev1.NodeAffinity `json:"nodeAffinity,omitempty"`

	// WorkloadAffinity gives the possibility to define affinity rules with other workloads running on Harvester.
	// +optional
	WorkloadAffinity *corev1.PodAffinity `json:"workloadAffinity,omitempty"`
}

type Volume struct {
	// VolumeType is the type of volume to attach.
	// Choose between: "storageClass" or "image"
	VolumeType VolumeType `json:"volumeType"`

	// ImageName is the name of the image to use if the volumeType is "image"
	// +optional
	ImageName string `json:"imageName,omitempty"`

	// StorageClass is the name of the storage class to be used if the volumeType is "storageClass"
	StorageClass string `json:"storageClass,omitempty"`

	// VolumeSize is the desired size of the volume. This satisfies to standard Kubernetes *resource.Quantity syntax.
	// Examples: 40.5Gi, 30M, etc. are valid
	// +optional
	VolumeSize *resource.Quantity `json:"volumeSize,omitempty"`

	// TODO: Implement a control in the admission webhook to check validity of BootOrders across volumes.
	// No duplicate values + exact sequence corresponding to total number of volumes

	// BootOrder is an integer that determines the order of priority of volumes for booting the VM
	// If absent, the sequence with which volumes appear in the manifest will be used.
	// +optional
	BootOrder int `json:"bootOrder,omitempty"`
}

// VolumeType is an enum string. It can only take the values: "storageClass" or "image"
// +kubebuilder:Validation:Enum:=storageClass,image
type VolumeType string

// HarvesterMachineStatus defines the observed state of HarvesterMachine
type HarvesterMachineStatus struct {
	// Ready is true when the provider resource is ready.
	Ready bool `json:"ready"`

	Conditions []capiv1beta1.Condition `json:"conditions,omitempty"`

	FailureReason  string                       `json:"failureReason,omitempty"`
	FailureMessage string                       `json:"failureMessage,omitempty"`
	Addresses      []capiv1beta1.MachineAddress `json:"addresses,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// HarvesterMachine is the Schema for the harvestermachines API
type HarvesterMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HarvesterMachineSpec   `json:"spec,omitempty"`
	Status HarvesterMachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HarvesterMachineList contains a list of HarvesterMachine
type HarvesterMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HarvesterMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HarvesterMachine{}, &HarvesterMachineList{})
}
