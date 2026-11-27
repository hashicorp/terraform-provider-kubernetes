// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"context"
	"testing"
	"time"

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
			"rollout": types.BoolType, "ready": types.BoolType, "fields": types.MapType{ElemType: types.StringType},
			"condition": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"type": types.StringType, "status": types.StringType}}},
		}, waitModel{Rollout: types.BoolNull(), Ready: types.BoolNull(), Fields: fields, Conditions: []conditionModel{}})
		expected, err := expectations(context.Background(), wait, identity{})
		if err != nil || len(expected) != 1 {
			t.Fatalf("%s: %v", path, err)
		}
		if met, _ := expected[0].met(obj); met != (pattern != "*") {
			t.Errorf("%s: met %t", path, met)
		}
	}
}

// Kinds without readiness rules of their own need a current Ready condition,
// and a status that names no generation counts only after a while.
func TestCurrentStatus(t *testing.T) {
	for name, tc := range map[string]struct {
		object string
		waited time.Duration
		want   bool
	}{
		"no status yet":             {`{"metadata":{"generation":1}}`, 0, false},
		"no status after a while":   {`{"metadata":{"generation":1}}`, settle, true},
		"ready for this generation": {`{"metadata":{"generation":2},"status":{"conditions":[{"type":"Ready","status":"True","observedGeneration":2}]}}`, 0, true},
		"ready for an older one":    {`{"metadata":{"generation":2},"status":{"conditions":[{"type":"Ready","status":"True","observedGeneration":1}]}}`, settle, false},
		"not ready":                 {`{"metadata":{"generation":1},"status":{"observedGeneration":1,"conditions":[{"type":"Ready","status":"False"}]}}`, settle, false},
		"generation in the status":  {`{"metadata":{"generation":1},"status":{"observedGeneration":1,"conditions":[{"type":"Ready","status":"True"}]}}`, 0, true},
	} {
		if got := currentStatus(&unstructured.Unstructured{Object: decodeJSON(t, tc.object)}, tc.waited); got != tc.want {
			t.Errorf("%s: got %t", name, got)
		}
	}
}
