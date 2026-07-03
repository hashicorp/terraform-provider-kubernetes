// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubemanifest

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ownedObj(obj map[string]interface{}, manager, fieldsV1 string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{Object: obj}
	u.SetManagedFields([]metav1.ManagedFieldsEntry{{
		Manager:    manager,
		Operation:  metav1.ManagedFieldsOperationApply,
		FieldsType: "FieldsV1",
		FieldsV1:   &metav1.FieldsV1{Raw: []byte(fieldsV1)},
	}})
	return u
}

func TestProjectOwned_ownsOnlyManagedScalars(t *testing.T) {
	obj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]interface{}{"name": "demo", "uid": "server-set"},
		"data":       map[string]interface{}{"a": "1", "b": "2"},
	}
	// We own only data.a.
	u := ownedObj(obj, "terraform", `{"f:data":{"f:a":{}}}`)
	got, err := ProjectOwned(u, "terraform", nil)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !strings.Contains(got, `"a":"1"`) || strings.Contains(got, `"b":"2"`) || strings.Contains(got, "server-set") {
		t.Fatalf("projection should contain only owned data.a, got %s", got)
	}
}

func TestProjectOwned_differentManagerEmpty(t *testing.T) {
	u := ownedObj(map[string]interface{}{"data": map[string]interface{}{"a": "1"}}, "other", `{"f:data":{"f:a":{}}}`)
	got, err := ProjectOwned(u, "terraform", nil)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got != "{}" {
		t.Fatalf("expected empty projection for a different manager, got %s", got)
	}
}

func TestDecodeYAML_jsonIsValidYAML(t *testing.T) {
	// JSON is a valid YAML document (YAML superset) — must decode fine.
	j := `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"demo"},"data":{"k":"v"}}`
	obj, err := DecodeYAML(j)
	if err != nil {
		t.Fatalf("JSON input should decode: %v", err)
	}
	if obj.GetKind() != "ConfigMap" || obj.GetName() != "demo" {
		t.Fatalf("unexpected: %s/%s", obj.GetKind(), obj.GetName())
	}
}

func TestDecodeYAML_multiDocRejected(t *testing.T) {
	_, err := DecodeYAML("kind: A\napiVersion: v1\nmetadata:\n  name: a\n---\nkind: B\napiVersion: v1\nmetadata:\n  name: b\n")
	if err == nil || !strings.Contains(err.Error(), "manifest_decode_multi") {
		t.Fatalf("multi-doc should be rejected with guidance, got: %v", err)
	}
}

func TestPathChanged(t *testing.T) {
	a := map[string]interface{}{"spec": map[string]interface{}{"x": "1"}}
	b := map[string]interface{}{"spec": map[string]interface{}{"x": "2"}}
	if !PathChanged(a, b, "spec.x") {
		t.Error("spec.x should be changed")
	}
	if PathChanged(a, a, "spec.x") {
		t.Error("identical should not be changed")
	}
	if PathChanged(a, a, "spec.missing") {
		t.Error("absent-in-both should not be changed")
	}
}

func TestIsImmutableErr(t *testing.T) {
	immutable := apierrors.NewInvalid(
		schema.GroupKind{Group: "apps", Kind: "StatefulSet"}, "web",
		field.ErrorList{field.Forbidden(field.NewPath("spec"), "updates to statefulset spec ... are forbidden")},
	)
	if !IsImmutableErr(immutable) {
		t.Error("immutable rejection should be detected")
	}
	if IsImmutableErr(apierrors.NewConflict(schema.GroupResource{Resource: "x"}, "n", nil)) {
		t.Error("conflict is not immutable")
	}
	if IsImmutableErr(nil) {
		t.Error("nil is not immutable")
	}
}

func TestApplyErrDiag_prefixesResource(t *testing.T) {
	summary, detail := ApplyErrDiag("kubernetes_manifest_patch", apierrors.NewConflict(schema.GroupResource{Resource: "deployments"}, "app", nil))
	if !strings.HasPrefix(summary, "kubernetes_manifest_patch:") || !strings.Contains(summary, "conflict") {
		t.Fatalf("unexpected summary: %q", summary)
	}
	if !strings.Contains(detail, "force_conflicts") {
		t.Fatalf("detail should guide to force_conflicts: %q", detail)
	}
}

func TestOpTimeout_floorAndDefault(t *testing.T) {
	tv := timeouts.Value{}
	// no floor → default
	got, d := OpTimeout(context.Background(), tv, "create", 0)
	if d.HasError() || got != DefaultOpTimeout {
		t.Fatalf("expected default %s, got %s (diags=%v)", DefaultOpTimeout, got, d)
	}
	// floor larger than default → floor
	got, _ = OpTimeout(context.Background(), tv, "create", 41*time.Minute)
	if got != 41*time.Minute {
		t.Fatalf("expected floor 41m, got %s", got)
	}
}
