// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"testing"
)

// TestInitContainerRestartPolicyValidation verifies that the restart_policy
// field on init_container accepts only "Always" (sidecar promotion) and rejects
// every other value. No cluster required — the ValidateFunc is exercised directly.
func TestInitContainerRestartPolicyValidation(t *testing.T) {
	s := containerFields(false)
	field, ok := s["restart_policy"]
	if !ok {
		t.Fatal("restart_policy field missing from containerFields schema")
	}
	if field.ValidateFunc == nil {
		t.Fatal("restart_policy ValidateFunc is nil")
	}

	validCases := []string{"Always"}
	for _, v := range validCases {
		_, es := field.ValidateFunc(v, "restart_policy")
		if len(es) > 0 {
			t.Errorf("expected %q to be valid, got errors: %v", v, es)
		}
	}

	invalidCases := []string{"Never", "OnFailure", "always", ""}
	for _, v := range invalidCases {
		_, es := field.ValidateFunc(v, "restart_policy")
		if len(es) == 0 {
			t.Errorf("expected %q to be invalid, but no errors were returned", v)
		}
	}
}
