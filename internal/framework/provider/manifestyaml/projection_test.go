// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestyaml

import (
	"encoding/json"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// makeObj builds an unstructured object with a single managedFields entry.
func makeObj(obj map[string]interface{}, manager, operation, subresource, fieldsV1 string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{Object: obj}
	u.SetManagedFields([]metav1.ManagedFieldsEntry{
		{
			Manager:     manager,
			Operation:   metav1.ManagedFieldsOperationType(operation),
			Subresource: subresource,
			FieldsType:  "FieldsV1",
			FieldsV1:    &metav1.FieldsV1{Raw: []byte(fieldsV1)},
		},
	})
	return u
}

func mustParse(t *testing.T, s string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	return m
}

func TestProjectOwned_dropsUnownedAndServerFields(t *testing.T) {
	obj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name": "app", "namespace": "default", "uid": "xxx", "creationTimestamp": "2020",
		},
		"data":   map[string]interface{}{"A": "1", "B": "2", "C": "3"}, // C added by someone else
		"status": map[string]interface{}{"phase": "Active"},           // server field
	}
	// We own metadata.name + data.A,B only.
	fields := `{"f:metadata":{"f:name":{}},"f:data":{"f:A":{},"f:B":{}}}`
	u := makeObj(obj, "terraform", "Apply", "", fields)

	got, err := projectOwned(u, "terraform", nil)
	if err != nil {
		t.Fatal(err)
	}
	want := mustParse(t, `{"data":{"A":"1","B":"2"},"metadata":{"name":"app"}}`)
	gotM := mustParse(t, got)
	if !jsonEqual(gotM, want) {
		t.Fatalf("projection mismatch:\n got=%s\nwant=%v", got, want)
	}
}

func TestProjectOwned_ignoreFieldsPruned(t *testing.T) {
	obj := map[string]interface{}{
		"apiVersion": "apps/v1", "kind": "Deployment",
		"metadata": map[string]interface{}{"name": "web", "namespace": "default"},
		"spec":     map[string]interface{}{"replicas": int64(2), "paused": false},
	}
	fields := `{"f:metadata":{"f:name":{}},"f:spec":{"f:replicas":{},"f:paused":{}}}`
	u := makeObj(obj, "terraform", "Apply", "", fields)

	got, err := projectOwned(u, "terraform", []string{"spec.replicas"})
	if err != nil {
		t.Fatal(err)
	}
	gotM := mustParse(t, got)
	spec, _ := gotM["spec"].(map[string]interface{})
	if _, present := spec["replicas"]; present {
		t.Fatalf("spec.replicas should have been pruned by ignore_fields, got=%s", got)
	}
	if _, present := spec["paused"]; !present {
		t.Fatalf("spec.paused should remain owned, got=%s", got)
	}
}

func TestProjectOwned_differentManagerYieldsEmpty(t *testing.T) {
	obj := map[string]interface{}{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]interface{}{"name": "app", "namespace": "default"},
		"data":     map[string]interface{}{"A": "1"},
	}
	fields := `{"f:data":{"f:A":{}}}`
	u := makeObj(obj, "kubectl", "Apply", "", fields) // owned by someone else

	got, err := projectOwned(u, "terraform", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "{}" {
		t.Fatalf("expected empty projection for unowned object, got=%s", got)
	}
}

func TestProjectOwned_ignoresUpdateAndStatusSubresource(t *testing.T) {
	obj := map[string]interface{}{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]interface{}{"name": "app", "namespace": "default"},
		"data":     map[string]interface{}{"A": "1"},
	}
	// Same manager name but Operation=Update (non-SSA) must be ignored.
	fields := `{"f:data":{"f:A":{}}}`
	u := makeObj(obj, "terraform", "Update", "", fields)

	got, err := projectOwned(u, "terraform", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "{}" {
		t.Fatalf("Update-operation fields must be ignored, got=%s", got)
	}
}

func TestRemoveDotPath(t *testing.T) {
	m := map[string]interface{}{
		"spec": map[string]interface{}{"replicas": 3, "template": map[string]interface{}{"x": 1}},
	}
	removeDotPath(m, "spec.replicas")
	spec := m["spec"].(map[string]interface{})
	if _, ok := spec["replicas"]; ok {
		t.Fatal("spec.replicas not removed")
	}
	if _, ok := spec["template"]; !ok {
		t.Fatal("spec.template should remain")
	}
	// missing path is a no-op
	removeDotPath(m, "spec.nonexistent.deep")
}

func jsonEqual(a, b map[string]interface{}) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}
