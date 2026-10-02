// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import "testing"

func TestJobHistoricalResourcesRejectMalformedCollections(t *testing.T) {
	for _, tc := range []struct {
		name      string
		pod       map[string]interface{}
		wantError bool
	}{
		{"omitted containers", map[string]interface{}{}, false},
		{"empty containers", map[string]interface{}{"container": []interface{}{}}, false},
		{"container map", map[string]interface{}{"container": map[string]interface{}{}}, true},
		{"init container map", map[string]interface{}{"init_container": map[string]interface{}{}}, true},
		{"null container element", map[string]interface{}{"container": []interface{}{nil}}, true},
		{"resources map", map[string]interface{}{"container": []interface{}{map[string]interface{}{"resources": map[string]interface{}{}}}}, true},
		{"resources repeated", map[string]interface{}{"container": []interface{}{map[string]interface{}{"resources": []interface{}{map[string]interface{}{}, map[string]interface{}{}}}}}, true},
		{"resources null element", map[string]interface{}{"init_container": []interface{}{map[string]interface{}{"resources": []interface{}{nil}}}}, true},
		{"empty resources", map[string]interface{}{"container": []interface{}{map[string]interface{}{"resources": []interface{}{}}}}, false},
		{"omitted resources", map[string]interface{}{"container": []interface{}{map[string]interface{}{}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := map[string]interface{}{"template": []interface{}{map[string]interface{}{
				"spec": []interface{}{tc.pod},
			}}}
			if err := upgradeJobResourcesV0(spec); (err != nil) != tc.wantError {
				t.Fatalf("conversion error = %v, want error %t", err, tc.wantError)
			}
		})
	}
	for _, malformed := range []interface{}{map[string]interface{}{}, []interface{}{nil}, []interface{}{map[string]interface{}{}, map[string]interface{}{}}} {
		spec := map[string]interface{}{"template": []interface{}{map[string]interface{}{"spec": malformed}}}
		if err := upgradeJobResourcesV0(spec); err == nil {
			t.Fatalf("malformed pod spec accepted: %#v", malformed)
		}
	}
}
