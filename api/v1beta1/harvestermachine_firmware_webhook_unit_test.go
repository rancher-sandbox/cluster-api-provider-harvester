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

func TestValidateMachineFirmware(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(m *HarvesterMachine)
		wantErr string
	}{
		{
			"efi alone is valid",
			func(m *HarvesterMachine) {
				m.Spec.Firmware = &Firmware{EFI: true}
			},
			"",
		},
		{
			"efi with secure boot is valid",
			func(m *HarvesterMachine) {
				m.Spec.Firmware = &Firmware{EFI: true, SecureBoot: true}
			},
			"",
		},
		{
			"secure boot without efi is rejected",
			func(m *HarvesterMachine) {
				m.Spec.Firmware = &Firmware{SecureBoot: true}
			},
			"secureBoot requires",
		},
		{
			"tpm alone is valid",
			func(m *HarvesterMachine) {
				m.Spec.TPM = &TPM{Enabled: true, Persistent: true}
			},
			"",
		},
	}
	for _, tc := range cases {
		m := validMachine()
		tc.mutate(m)

		_, err := validateHarvesterMachine(m)

		if tc.wantErr == "" && err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
		}

		if tc.wantErr != "" && err == nil {
			t.Errorf("%s: expected an error", tc.name)
		}

		if tc.wantErr != "" && err != nil && !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("%s: error should mention %q: %v", tc.name, tc.wantErr, err)
		}
	}
}
