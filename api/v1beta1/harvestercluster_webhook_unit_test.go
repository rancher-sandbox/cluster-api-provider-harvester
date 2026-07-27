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

package v1beta1

import (
	"strings"
	"testing"
)

func validCluster() *HarvesterCluster {
	return &HarvesterCluster{
		Spec: HarvesterClusterSpec{
			TargetNamespace:    "default",
			IdentitySecret:     SecretKey{Namespace: "default", Name: "id"},
			LoadBalancerConfig: LoadBalancerConfig{IPAMType: IPAMType(DHCP)},
		},
	}
}

func TestValidateVMNetworkConfigPoolSources(t *testing.T) {
	cases := []struct {
		name    string
		cfg     *VMNetworkConfig
		wantErr bool
	}{
		{"single ipPoolRef", &VMNetworkConfig{IPPoolRef: "pool-a", Gateway: "10.0.0.1", SubnetMask: "255.255.255.0"}, false},
		{"ipPoolRefs list only", &VMNetworkConfig{IPPoolRefs: []string{"pool-a", "pool-b"}, Gateway: "10.0.0.1", SubnetMask: "255.255.255.0"}, false},
		{"no pool source", &VMNetworkConfig{Gateway: "10.0.0.1", SubnetMask: "255.255.255.0"}, true},
	}
	for _, tc := range cases {
		c := validCluster()
		c.Spec.VMNetworkConfig = tc.cfg

		_, err := validateHarvesterCluster(c)

		if tc.wantErr && err == nil {
			t.Errorf("%s: expected an error", tc.name)
		}

		if !tc.wantErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
		}

		if tc.wantErr && err != nil && !strings.Contains(err.Error(), "ipPoolRefs") {
			t.Errorf("%s: error should mention ipPoolRefs: %v", tc.name, err)
		}
	}
}
