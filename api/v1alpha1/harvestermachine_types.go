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
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HarvesterMachineSpec defines the desired state of HarvesterMachine
type HarvesterMachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of HarvesterMachine. Edit harvestermachine_types.go to remove/update
	ProviderID    string `json:"providerID"`
	FailureDomain string `json:"failureDomain,omitempty"`
}

// HarvesterMachineStatus defines the observed state of HarvesterMachine
type HarvesterMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Ready string `json:"ready"`

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
