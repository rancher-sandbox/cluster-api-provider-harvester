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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HarvesterMachineTemplateSpec defines the desired state of HarvesterMachineTemplate.
type HarvesterMachineTemplateSpec struct {
	// Template is the HarvesterMachineTemplate template
	Template HarvesterMachineTemplateResource `json:"template,omitempty"`
}

// HarvesterMachineTemplateResource describes the data needed to create a HarvesterMachine from a template.
type HarvesterMachineTemplateResource struct {
	// Spec is the specification of the desired behavior of the machine.
	Spec HarvesterMachineSpec `json:"spec"`
}

//+kubebuilder:object:root=true

// HarvesterMachineTemplate is the Schema for the harvestermachinetemplates API.
type HarvesterMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec HarvesterMachineTemplateSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// HarvesterMachineTemplateList contains a list of HarvesterMachineTemplate.
type HarvesterMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []HarvesterMachineTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HarvesterMachineTemplate{}, &HarvesterMachineTemplateList{})
}
