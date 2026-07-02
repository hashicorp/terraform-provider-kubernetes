// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestyaml

import (
	"encoding/json"
	"strings"
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
		"status": map[string]interface{}{"phase": "Active"},            // server field
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

// --- golden / hardening tests ----------------------------------------------

// Associative lists: with schemaless (deduced) typing, lists are ATOMIC — owning any
// element means the whole list is projected. Fine-grained per-element (associative-key)
// extraction requires the real OpenAPI schema and is a documented MVP limitation
// (RFC-011 §6.3.1). This test pins that behavior so the boundary is explicit.
func TestProjectOwned_listsAreAtomic_knownLimitation(t *testing.T) {
	obj := map[string]interface{}{
		"apiVersion": "apps/v1", "kind": "Deployment",
		"metadata": map[string]interface{}{"name": "web", "namespace": "default"},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{"name": "app", "image": "nginx:1.27"},
						map[string]interface{}{"name": "sidecar", "image": "envoy:1.30"},
					},
				},
			},
		},
	}
	fields := `{"f:spec":{"f:template":{"f:spec":{"f:containers":{"k:{\"name\":\"app\"}":{"f:name":{},"f:image":{}}}}}}}`
	u := makeObj(obj, "terraform", "Apply", "", fields)

	got, err := projectOwned(u, "terraform", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Atomic-list behavior: the whole containers list is projected (both app + sidecar).
	if !strings.Contains(got, `"name":"app"`) {
		t.Fatalf("expected owned container list, got=%s", got)
	}
	if !strings.Contains(got, "sidecar") {
		t.Fatalf("documented limitation: schemaless lists are atomic → whole list projected; got=%s", got)
	}
}

// Arbitrary CR / preserve-unknown-fields shapes must not panic and should project owned keys.
func TestProjectOwned_customResourceShape(t *testing.T) {
	obj := map[string]interface{}{
		"apiVersion": "example.com/v1", "kind": "Widget",
		"metadata": map[string]interface{}{"name": "w1", "namespace": "default"},
		"spec": map[string]interface{}{
			"nested": map[string]interface{}{"free": map[string]interface{}{"any": "thing", "n": float64(3)}},
			"list":   []interface{}{"a", "b"},
		},
	}
	fields := `{"f:spec":{"f:nested":{"f:free":{"f:any":{}}}}}`
	u := makeObj(obj, "terraform", "Apply", "", fields)

	got, err := projectOwned(u, "terraform", nil)
	if err != nil {
		t.Fatalf("must not error on arbitrary CR shape: %v", err)
	}
	if !strings.Contains(got, `"any":"thing"`) {
		t.Fatalf("expected owned nested field, got=%s", got)
	}
	if strings.Contains(got, `"list"`) {
		t.Fatalf("unowned spec.list must be excluded, got=%s", got)
	}
}

// First-apply race: an object without our managedFields yet projects to "{}".
func TestProjectOwned_firstApplyEmpty(t *testing.T) {
	obj := map[string]interface{}{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]interface{}{"name": "app", "namespace": "default"},
		"data":     map[string]interface{}{"A": "1"},
	}
	u := &unstructured.Unstructured{Object: obj} // no managedFields at all
	got, err := projectOwned(u, "terraform", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "{}" {
		t.Fatalf("no managedFields → empty projection, got=%s", got)
	}
}

// Canonical stability: projecting the same inputs twice yields byte-identical output
// (so apply-result vs dry-run comparisons are stable).
func TestProjectOwned_canonicalStable(t *testing.T) {
	obj := map[string]interface{}{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]interface{}{"name": "app", "namespace": "default"},
		"data":     map[string]interface{}{"Z": "26", "A": "1", "M": "13"},
	}
	fields := `{"f:metadata":{"f:name":{}},"f:data":{"f:A":{},"f:M":{},"f:Z":{}}}`
	u := makeObj(obj, "terraform", "Apply", "", fields)

	a, err := projectOwned(u, "terraform", nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := projectOwned(u, "terraform", nil)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("projection not stable:\n a=%s\n b=%s", a, b)
	}
	// keys must be sorted (canonical): A before M before Z.
	if strings.Index(a, `"A"`) > strings.Index(a, `"M"`) || strings.Index(a, `"M"`) > strings.Index(a, `"Z"`) {
		t.Fatalf("keys not canonical/sorted: %s", a)
	}
}

func TestNormalizeYAML_ignoresCosmetics(t *testing.T) {
	a := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: app\ndata:\n  a: \"1\"\n  b: \"2\"\n"
	// same doc: reordered keys, comment, extra whitespace, flow style
	b := "# a comment\nkind: ConfigMap\ndata: {b: \"2\", a: \"1\"}\napiVersion: v1\nmetadata: {name: app}\n"
	na, err := normalizeYAML(a)
	if err != nil {
		t.Fatal(err)
	}
	nb, err := normalizeYAML(b)
	if err != nil {
		t.Fatal(err)
	}
	if na != nb {
		t.Fatalf("cosmetically-equal YAML should normalize equal:\n na=%s\n nb=%s", na, nb)
	}
	// a real change must differ
	c, _ := normalizeYAML("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: app\ndata:\n  a: \"CHANGED\"\n")
	if na == c {
		t.Fatal("semantic change must normalize differently")
	}
}
