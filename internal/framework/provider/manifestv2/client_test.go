// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Only errors that a resource applied in parallel can resolve are retried.
func TestAwaitsDependency(t *testing.T) {
	configMaps := schema.GroupResource{Resource: "configmaps"}
	for name, tc := range map[string]struct {
		err  error
		want bool
	}{
		"kind not served yet":          {unknownKindError{"example.com/v1", "Widget"}, true},
		"no such built-in kind":        {unknownKindError{"apps/v1", "Deploymnet"}, false},
		"namespace missing":            {apierrors.NewNotFound(schema.GroupResource{Resource: "namespaces"}, "prod"), true},
		"resource not served yet":      {&apierrors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound, Code: 404}}, true},
		"the object itself is missing": {apierrors.NewNotFound(configMaps, "settings"), false},
		"conflict":                     {apierrors.NewConflict(configMaps, "settings", nil), false},
		"rejected":                     {apierrors.NewBadRequest("unknown field"), false},
	} {
		if _, got := awaitsDependency(tc.err); got != tc.want {
			t.Errorf("%s: got %t, want %t", name, got, tc.want)
		}
	}
}

// Numbers and bools become strings where the schema has a string, and only there.
func TestCoerce(t *testing.T) {
	var document openAPIDocument
	if err := json.Unmarshal([]byte(`{"components":{"schemas":{
		"cm": {"type":"object","properties":{"data":{"type":"object","additionalProperties":{"type":"string"}},
			"metadata":{"allOf":[{"$ref":"#/components/schemas/meta"}]},"port":{"x-kubernetes-int-or-string":true,"anyOf":[{"type":"integer"},{"type":"string"}]},
			"cpu":{"oneOf":[{"type":"string"},{"type":"number"}]}}},
		"meta": {"type":"object","properties":{"labels":{"type":"object","additionalProperties":{"type":"string"}}}}}}}`), &document); err != nil {
		t.Fatal(err)
	}
	value := map[string]any{"data": map[string]any{"PORT": int64(8080), "DEBUG": true, "RATIO": 0.5}, "metadata": map[string]any{"labels": map[string]any{"version": int64(2)}},
		"port": int64(80), "cpu": int64(1)}
	changed := false
	coerce(value, document.Components.Schemas["cm"], document.Components.Schemas, 0, &changed)
	want := map[string]any{"data": map[string]any{"PORT": "8080", "DEBUG": "true", "RATIO": "0.5"}, "metadata": map[string]any{"labels": map[string]any{"version": "2"}},
		"port": int64(80), "cpu": int64(1)}
	if !changed || !reflect.DeepEqual(value, want) {
		t.Fatalf("got %v", value)
	}
}

func TestWithDriftNote(t *testing.T) {
	for _, body := range []string{"kind: A", "kind: A\n"} {
		noted := withDriftNote(body, []string{"data.a", "spec.b"})
		if !strings.Contains(noted, "A\n# Changed outside Terraform: data.a, spec.b") || withDriftNote(noted, nil) != body || withDriftNote(withDriftNote(noted, []string{"x"}), nil) != body {
			t.Errorf("%q: %q", body, noted)
		}
	}
}
