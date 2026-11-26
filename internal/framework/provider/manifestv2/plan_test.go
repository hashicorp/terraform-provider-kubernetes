// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import "testing"

// Only the API version may change without making it another object.
func TestSameObject(t *testing.T) {
	prior := identity{"apps/v1", "Deployment", "staging", "web"}
	for name, tc := range map[string]struct {
		prior, declared identity
		want            bool
	}{
		"unchanged":                      {prior, prior, true},
		"newer API version":              {prior, identity{"apps/v2", "Deployment", "staging", "web"}, true},
		"renamed":                        {prior, identity{"apps/v1", "Deployment", "staging", "api"}, false},
		"another namespace":              {prior, identity{"apps/v1", "Deployment", "prod", "web"}, false},
		"namespace removed":              {prior, identity{"apps/v1", "Deployment", "", "web"}, false},
		"namespace never written":        {identity{"apps/v1", "Deployment", "default", "web"}, identity{"apps/v1", "Deployment", "", "web"}, true},
		"cluster-scoped":                 {identity{"v1", "Namespace", "", "prod"}, identity{"v1", "Namespace", "", "prod"}, true},
		"another group with a like kind": {prior, identity{"example.com/v1", "Deployment", "staging", "web"}, false},
	} {
		if got := sameObject(tc.prior, tc.declared); got != tc.want {
			t.Errorf("%s: got %t, want %t", name, got, tc.want)
		}
	}
}
