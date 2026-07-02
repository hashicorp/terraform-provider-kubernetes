// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestyaml

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestDecodeYAML_singleDocOK(t *testing.T) {
	y := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: demo
  namespace: default
data:
  a: b
`
	obj, err := decodeYAML(y)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obj.GetKind() != "ConfigMap" || obj.GetName() != "demo" {
		t.Fatalf("unexpected object: %s/%s", obj.GetKind(), obj.GetName())
	}
}

func TestDecodeYAML_leadingSeparatorSingleDocOK(t *testing.T) {
	// A single document that merely starts with "---" is valid.
	y := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: demo
`
	if _, err := decodeYAML(y); err != nil {
		t.Fatalf("leading '---' single doc should be valid, got: %v", err)
	}
}

func TestDecodeYAML_multiDocRejected(t *testing.T) {
	y := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: a
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: b
`
	_, err := decodeYAML(y)
	if err == nil {
		t.Fatal("expected multi-document YAML to be rejected")
	}
	if !strings.Contains(err.Error(), "manifest_decode_multi") {
		t.Fatalf("error should point to manifest_decode_multi, got: %v", err)
	}
}

func TestCountYAMLDocuments(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"single", "kind: A\n", 1},
		{"leading-sep", "---\nkind: A\n", 1},
		{"trailing-sep", "kind: A\n---\n", 1},
		{"two", "kind: A\n---\nkind: B\n", 2},
		{"three-with-blank", "kind: A\n---\n\n---\nkind: B\n---\nkind: C\n", 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n, err := countYAMLDocuments(c.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if n != c.want {
				t.Fatalf("got %d docs, want %d", n, c.want)
			}
		})
	}
}

func TestPathChanged(t *testing.T) {
	base := map[string]interface{}{
		"spec": map[string]interface{}{
			"serviceName": "web",
			"replicas":    int64(3),
			"volumeClaimTemplates": []interface{}{
				map[string]interface{}{"metadata": map[string]interface{}{"name": "data"}},
			},
		},
	}
	same := map[string]interface{}{
		"spec": map[string]interface{}{
			"serviceName": "web",
			"replicas":    int64(3),
			"volumeClaimTemplates": []interface{}{
				map[string]interface{}{"metadata": map[string]interface{}{"name": "data"}},
			},
		},
	}
	changed := map[string]interface{}{
		"spec": map[string]interface{}{
			"serviceName": "web2", // changed
			"replicas":    int64(3),
		},
	}

	if pathChanged(base, same, "spec.serviceName") {
		t.Error("serviceName should be unchanged")
	}
	if !pathChanged(base, changed, "spec.serviceName") {
		t.Error("serviceName should be detected as changed")
	}
	// present in base, absent in changed → changed
	if !pathChanged(base, changed, "spec.volumeClaimTemplates") {
		t.Error("removed volumeClaimTemplates should count as changed")
	}
	// path missing in both → not changed
	if pathChanged(base, same, "spec.updateStrategy.type") {
		t.Error("absent-in-both path should not be changed")
	}
	// traversal through a non-map segment → treated as absent, not a panic
	if pathChanged(base, same, "spec.serviceName.nope") {
		t.Error("descending into a scalar should be absent-in-both, not changed")
	}
}

func TestValueAtDotPath(t *testing.T) {
	m := map[string]interface{}{
		"a": map[string]interface{}{"b": map[string]interface{}{"c": "x"}},
	}
	if v, ok := valueAtDotPath(m, "a.b.c"); !ok || v != "x" {
		t.Fatalf("got %v ok=%v, want x", v, ok)
	}
	if _, ok := valueAtDotPath(m, "a.b.z"); ok {
		t.Fatal("missing leaf should return ok=false")
	}
	if _, ok := valueAtDotPath(m, "a.b.c.d"); ok {
		t.Fatal("descending past a scalar should return ok=false")
	}
}

func TestIsImmutableErr(t *testing.T) {
	// Realistic StatefulSet immutable-spec rejection: a typed Invalid (422) error whose
	// message contains "... are forbidden", exactly as the API server returns it.
	realistic := apierrors.NewInvalid(
		schema.GroupKind{Group: "apps", Kind: "StatefulSet"}, "web",
		field.ErrorList{
			field.Forbidden(field.NewPath("spec"),
				"updates to statefulset spec for fields other than 'replicas', 'template', "+
					"'updateStrategy', 'persistentVolumeClaimRetentionPolicy' and 'minReadySeconds' are forbidden"),
		},
	)
	// A typed Invalid error without an immutability signal (e.g. a plain validation error).
	plainInvalid := apierrors.NewInvalid(
		schema.GroupKind{Group: "", Kind: "ConfigMap"}, "cm",
		field.ErrorList{field.Required(field.NewPath("data"), "must be provided")},
	)
	conflict := apierrors.NewConflict(schema.GroupResource{Group: "apps", Resource: "statefulsets"}, "web", fmt.Errorf("conflict"))
	notFound := apierrors.NewNotFound(schema.GroupResource{Group: "apps", Resource: "statefulsets"}, "web")

	if isImmutableErr(nil) {
		t.Error("nil should not be immutable")
	}
	if !isImmutableErr(realistic) {
		t.Errorf("realistic 'are forbidden' message should be immutable: %v", realistic)
	}
	if isImmutableErr(plainInvalid) {
		t.Error("a plain validation Invalid error should not be classified as immutable")
	}
	if isImmutableErr(conflict) {
		t.Error("conflict should not be classified as immutable")
	}
	if isImmutableErr(notFound) {
		t.Error("not-found should not be classified as immutable")
	}
}

func TestApplyErrDiag_conflict(t *testing.T) {
	conflict := apierrors.NewConflict(schema.GroupResource{Group: "apps", Resource: "statefulsets"}, "web", fmt.Errorf("Apply failed with 1 conflict"))
	summary, detail := applyErrDiag(conflict)
	if !strings.Contains(summary, "conflict") {
		t.Errorf("summary should mention conflict, got %q", summary)
	}
	if !strings.Contains(detail, "force_conflicts") {
		t.Errorf("detail should guide to force_conflicts, got %q", detail)
	}
}

func TestApplyErrDiag_generic(t *testing.T) {
	summary, detail := applyErrDiag(fmt.Errorf("some transport error"))
	if !strings.Contains(summary, "apply failed") {
		t.Errorf("generic summary unexpected: %q", summary)
	}
	if !strings.Contains(detail, "transport error") {
		t.Errorf("generic detail should carry the error: %q", detail)
	}
}

func TestOpTimeout_neverShorterThanWait(t *testing.T) {
	// wait timeout larger than the default op timeout should widen the op timeout.
	w := &waitModel{
		Rollout: types.BoolValue(true),
		Timeout: types.StringValue("40m"),
	}
	// A null timeouts.Value → Create() returns the supplied default (20m). Wait is 40m,
	// so the effective op timeout must be >= 41m.
	tv := timeouts.Value{}
	got, diags := opTimeout(context.Background(), tv, "create", w)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got < 41*time.Minute {
		t.Fatalf("op timeout %s should be >= wait+1m (41m)", got)
	}
}

func TestOpTimeout_defaultWhenNoWait(t *testing.T) {
	tv := timeouts.Value{}
	got, diags := opTimeout(context.Background(), tv, "delete", nil)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got != defaultOpTimeout {
		t.Fatalf("expected default op timeout %s, got %s", defaultOpTimeout, got)
	}
}
