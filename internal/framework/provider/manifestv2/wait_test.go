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
		expected := waitFor(t, waitModel{Fields: fieldsWait(path, pattern)}, identity{})
		if met, _, _ := expected[0].met(obj); met != (pattern != "*") {
			t.Errorf("%s: met %t", path, met)
		}
	}
}

// A wait that ends first says where the object stood.
func TestWaitStatus(t *testing.T) {
	pvc := &unstructured.Unstructured{Object: decodeJSON(t, `{"apiVersion":"v1","kind":"PersistentVolumeClaim",
		"metadata":{"name":"data"},"status":{"phase":"Pending"}}`)}
	deployment := &unstructured.Unstructured{Object: decodeJSON(t, `{"apiVersion":"apps/v1","kind":"Deployment",
		"metadata":{"name":"web","generation":2},"spec":{"replicas":2},
		"status":{"observedGeneration":2,"replicas":2,"updatedReplicas":1}}`)}
	widget := &unstructured.Unstructured{Object: decodeJSON(t, `{"apiVersion":"example.com/v1","kind":"Widget",
		"metadata":{"name":"w","generation":1},"status":{"conditions":[{"type":"Ready","status":"False","reason":"Pending","message":"no endpoint"}]}}`)}
	secret := &unstructured.Unstructured{Object: decodeJSON(t, `{"apiVersion":"v1","kind":"Secret","metadata":{"name":"s"},"data":{"token":"c2VjcmV0"}}`)}
	ready := func(kind string) []conditionModel {
		return []conditionModel{{Type: types.StringValue(kind), Status: types.StringNull()}}
	}
	for name, tc := range map[string]struct {
		wait waitModel
		id   identity
		obj  *unstructured.Unstructured
		want string
	}{
		"ready":                {waitModel{Ready: types.BoolValue(true)}, identity{}, pvc, "PVC is not Bound. phase: Pending"},
		"ready, custom kind":   {waitModel{Ready: types.BoolValue(true)}, identity{}, widget, "no endpoint"},
		"rollout":              {waitModel{Rollout: types.BoolValue(true)}, identity{APIVersion: "apps/v1", Kind: "Deployment"}, deployment, `Waiting for deployment "web" rollout to finish: 1 out of 2 new replicas have been updated...`},
		"condition":            {waitModel{Conditions: ready("Resizing")}, identity{}, pvc, "no Resizing condition"},
		"condition with state": {waitModel{Conditions: ready("Ready")}, identity{}, widget, "Ready is False (Pending: no endpoint)"},
		"field":                {waitModel{Fields: fieldsWait("status.phase", "Bound")}, identity{}, pvc, `status.phase is "Pending"`},
		"field not set":        {waitModel{Fields: fieldsWait("status.capacity", "*")}, identity{}, pvc, "status.capacity is not set"},
		"field of a secret":    {waitModel{Fields: fieldsWait("data.token", "x")}, identity{}, secret, "data.token does not match"},
	} {
		if met, seen, _ := waitFor(t, tc.wait, tc.id)[0].met(tc.obj); met || seen != tc.want {
			t.Errorf("%s: met %t, status %q", name, met, seen)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	<-ctx.Done()
	if got := stopped(ctx, "the object to be ready", "PVC is not Bound. phase: Pending", nil).Error(); got != "timed out waiting for the object to be ready; last status: PVC is not Bound. phase: Pending" {
		t.Errorf("message %q", got)
	}
}

func fieldsWait(path, pattern string) types.Map {
	fields, _ := types.MapValueFrom(context.Background(), types.StringType, map[string]string{path: pattern})
	return fields
}

// waitFor builds the expectations of a wait block; unset arguments are null.
func waitFor(t *testing.T, w waitModel, id identity) []expectation {
	t.Helper()
	if w.Rollout == (types.Bool{}) {
		w.Rollout = types.BoolNull()
	}
	if w.Ready == (types.Bool{}) {
		w.Ready = types.BoolNull()
	}
	if w.Fields.IsNull() || len(w.Fields.Elements()) == 0 {
		w.Fields = types.MapNull(types.StringType)
	}
	if w.Conditions == nil {
		w.Conditions = []conditionModel{}
	}
	wait, diags := types.ObjectValueFrom(context.Background(), map[string]attr.Type{
		"rollout": types.BoolType, "ready": types.BoolType, "fields": types.MapType{ElemType: types.StringType},
		"condition": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"type": types.StringType, "status": types.StringType}}},
	}, w)
	expected, err := expectations(context.Background(), wait, id)
	if diags.HasError() || err != nil || len(expected) != 1 {
		t.Fatalf("wait %v: %v %v", w, diags, err)
	}
	return expected
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
