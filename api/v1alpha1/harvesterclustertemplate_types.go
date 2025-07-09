/*
Copyright 2025.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// / HarvesterClusterTemplateSpec defines the desired state of HarvesterClusterTemplate.
type HarvesterClusterTemplateSpec struct {
	Template HarvesterClusterTemplateResource `json:"template"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=harvesterclustertemplates,scope=Namespaced,categories=cluster-api,shortName=hvct
// +kubebuilder:storageversion

// HarvesterClusterTemplate is the Schema for the infraclustertemplates API.
type HarvesterClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the specification for the template resource
	Spec HarvesterClusterTemplateSpec `json:"spec,omitempty"`
	// Status is the status of the template HarvesterCluster resource
	Status HarvesterClusterStatus `json:"status,omitempty"`
}

type HarvesterClusterTemplateResource struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata.
	// +optional
	ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty"`
	Spec       HarvesterClusterSpec `json:"spec"`
}

//+kubebuilder:object:root=true

// HarvesterClusterTemplateList contains a list of HarvesterClusterTemplates.
type HarvesterClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HarvesterClusterTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HarvesterClusterTemplate{}, &HarvesterClusterTemplateList{})
}
