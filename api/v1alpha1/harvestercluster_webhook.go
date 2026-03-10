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

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"k8s.io/apimachinery/pkg/runtime"
)

// HarvesterClusterValidator implements admission.CustomValidator for HarvesterCluster.
type HarvesterClusterValidator struct{}

// SetupHarvesterClusterWebhookWithManager sets up the validating webhook for HarvesterCluster.
func SetupHarvesterClusterWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&HarvesterCluster{}).
		WithValidator(&HarvesterClusterValidator{}).
		Complete()
}

//nolint:lll
// +kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1alpha1-harvestercluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=harvesterclusters,verbs=create;update,versions=v1alpha1,name=vharvestercluster.kb.io,admissionReviewVersions=v1

var _ admission.CustomValidator = &HarvesterClusterValidator{}

// ValidateCreate implements admission.CustomValidator.
func (v *HarvesterClusterValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	c, ok := obj.(*HarvesterCluster)
	if !ok {
		return nil, fmt.Errorf("expected HarvesterCluster, got %T", obj)
	}

	return validateHarvesterCluster(c)
}

// ValidateUpdate implements admission.CustomValidator.
func (v *HarvesterClusterValidator) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	c, ok := newObj.(*HarvesterCluster)
	if !ok {
		return nil, fmt.Errorf("expected HarvesterCluster, got %T", newObj)
	}

	return validateHarvesterCluster(c)
}

// ValidateDelete implements admission.CustomValidator.
func (v *HarvesterClusterValidator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateHarvesterCluster(r *HarvesterCluster) (admission.Warnings, error) {
	var errs []string

	if r.Spec.TargetNamespace == "" {
		errs = append(errs, "spec.targetNamespace is required")
	}

	if r.Spec.IdentitySecret.Name == "" {
		errs = append(errs, "spec.identitySecret.name is required")
	}

	if r.Spec.IdentitySecret.Namespace == "" {
		errs = append(errs, "spec.identitySecret.namespace is required")
	}

	if r.Spec.LoadBalancerConfig.IPAMType != IPAMType(DHCP) && r.Spec.LoadBalancerConfig.IPAMType != IPAMType(POOL) {
		errs = append(errs, fmt.Sprintf("spec.loadBalancerConfig.ipamType must be %q or %q", DHCP, POOL))
	}

	if r.Spec.VMNetworkConfig != nil {
		vmCfg := r.Spec.VMNetworkConfig
		if vmCfg.IPPoolRef == "" && vmCfg.IPPool == nil {
			errs = append(errs, "spec.vmNetworkConfig requires either ipPoolRef or ipPool")
		}

		if vmCfg.Gateway == "" {
			errs = append(errs, "spec.vmNetworkConfig.gateway is required")
		} else if net.ParseIP(vmCfg.Gateway) == nil {
			errs = append(errs, fmt.Sprintf("spec.vmNetworkConfig.gateway %q is not a valid IP address", vmCfg.Gateway))
		}

		if vmCfg.SubnetMask == "" {
			errs = append(errs, "spec.vmNetworkConfig.subnetMask is required")
		} else if net.ParseIP(vmCfg.SubnetMask) == nil {
			errs = append(errs, fmt.Sprintf("spec.vmNetworkConfig.subnetMask %q is not a valid IP address", vmCfg.SubnetMask))
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("validation failed for HarvesterCluster %s/%s: %s",
			r.Namespace, r.Name, strings.Join(errs, "; "))
	}

	return nil, nil
}
