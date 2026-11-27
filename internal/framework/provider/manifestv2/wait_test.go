// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Field paths are written as in kubernetes_manifest.
func TestWaitFields(t *testing.T) {
	obj := &unstructured.Unstructured{Object: decodeJSON(t, `{"metadata":{"annotations":{"example.com/owner":"team-a"}},
		"status":{"phase":"Running","message":null,"containerStatuses":[{"ready":true}]}}`)}
	for path, pattern := range map[string]string{
		"status.phase":                              "^Running",
		"status.containerStatuses[0].ready":         "true",
		`metadata.annotations["example.com/owner"]`: "team-a",
		"status.podIP":                              "",
		"status.message":                            "",
	} {
		if pattern == "" {
			pattern = "*"
		} else {
			pattern += "$"
		}
		fields, _ := types.MapValueFrom(context.Background(), types.StringType, map[string]string{path: pattern})
		wait, _ := types.ObjectValueFrom(context.Background(), map[string]attr.Type{
			"rollout": types.BoolType, "fields": types.MapType{ElemType: types.StringType},
			"condition": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"type": types.StringType, "status": types.StringType}}},
		}, waitModel{Rollout: types.BoolNull(), Fields: fields, Conditions: []conditionModel{}})
		expected, err := expectations(context.Background(), wait, identity{})
		if err != nil || len(expected) != 1 {
			t.Fatalf("%s: %v", path, err)
		}
		if met, _ := expected[0].met(obj); met != (pattern != "*") {
			t.Errorf("%s: met %t", path, met)
		}
	}
}
