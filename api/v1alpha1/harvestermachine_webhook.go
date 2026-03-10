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
	"context"
	"fmt"
	"net"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// HarvesterMachineValidator implements admission.CustomValidator for HarvesterMachine.
type HarvesterMachineValidator struct{}

// SetupWebhookWithManager sets up the validating webhook for HarvesterMachine.
func SetupHarvesterMachineWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&HarvesterMachine{}).
		WithValidator(&HarvesterMachineValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1alpha1-harvestermachine,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=harvestermachines,verbs=create;update,versions=v1alpha1,name=vharvestermachine.kb.io,admissionReviewVersions=v1

var _ admission.CustomValidator = &HarvesterMachineValidator{}

// ValidateCreate implements admission.CustomValidator.
func (v *HarvesterMachineValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	m, ok := obj.(*HarvesterMachine)
	if !ok {
		return nil, fmt.Errorf("expected HarvesterMachine, got %T", obj)
	}
	return validateHarvesterMachine(m)
}

// ValidateUpdate implements admission.CustomValidator.
func (v *HarvesterMachineValidator) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	m, ok := newObj.(*HarvesterMachine)
	if !ok {
		return nil, fmt.Errorf("expected HarvesterMachine, got %T", newObj)
	}
	return validateHarvesterMachine(m)
}

// ValidateDelete implements admission.CustomValidator.
func (v *HarvesterMachineValidator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateHarvesterMachine(r *HarvesterMachine) (admission.Warnings, error) {
	var errs []string

	if r.Spec.CPU == 0 {
		errs = append(errs, "spec.cpu must be greater than 0")
	}

	if r.Spec.Memory == "" {
		errs = append(errs, "spec.memory is required")
	} else if _, err := resource.ParseQuantity(r.Spec.Memory); err != nil {
		errs = append(errs, fmt.Sprintf("spec.memory %q is not a valid resource quantity: %v", r.Spec.Memory, err))
	}

	if r.Spec.SSHUser == "" {
		errs = append(errs, "spec.sshUser is required")
	}

	if r.Spec.SSHKeyPair == "" {
		errs = append(errs, "spec.sshKeyPair is required")
	}

	if len(r.Spec.Volumes) == 0 {
		errs = append(errs, "spec.volumes must contain at least one volume")
	}

	for i, vol := range r.Spec.Volumes {
		if vol.VolumeType != "image" && vol.VolumeType != "storageClass" {
			errs = append(errs, fmt.Sprintf("spec.volumes[%d].volumeType must be 'image' or 'storageClass'", i))
		}
		if vol.VolumeType == "image" && vol.ImageName == "" {
			errs = append(errs, fmt.Sprintf("spec.volumes[%d].imageName is required when volumeType is 'image'", i))
		}
		if vol.VolumeType == "storageClass" && vol.StorageClass == "" {
			errs = append(errs, fmt.Sprintf("spec.volumes[%d].storageClass is required when volumeType is 'storageClass'", i))
		}
	}

	if len(r.Spec.Networks) == 0 {
		errs = append(errs, "spec.networks must contain at least one network")
	}

	if r.Spec.NetworkConfig != nil {
		if r.Spec.NetworkConfig.Address == "" {
			errs = append(errs, "spec.networkConfig.address is required when networkConfig is set")
		}
		if r.Spec.NetworkConfig.Gateway == "" {
			errs = append(errs, "spec.networkConfig.gateway is required when networkConfig is set")
		} else if net.ParseIP(r.Spec.NetworkConfig.Gateway) == nil {
			errs = append(errs, fmt.Sprintf("spec.networkConfig.gateway %q is not a valid IP address", r.Spec.NetworkConfig.Gateway))
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("validation failed for HarvesterMachine %s/%s: %s",
			r.Namespace, r.Name, strings.Join(errs, "; "))
	}

	return nil, nil
}
