// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

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

// An apply that starts after its namespace or kind has been missing for a
// minute fails at once, says what is missing and keeps the API error.
func TestRetryWhileMissing(t *testing.T) {
	id := identity{APIVersion: "example.com/v1", Kind: "Widget", Namespace: "prod", Name: "w"}
	for dependency, tc := range map[string]struct {
		err   error
		check string
	}{
		"namespace prod":             {apierrors.NewNotFound(schema.GroupResource{Resource: "namespaces"}, "prod"), "check the namespace name."},
		"kind example.com/v1 Widget": {unknownKindError{"example.com/v1", "Widget"}, "check apiVersion and kind."},
	} {
		missingSince.Store(dependency, time.Now().Add(-2*time.Minute))
		err := retryWhile(context.Background(), id, func() error { return tc.err })
		missingSince.Delete(dependency)
		if !errors.Is(err, tc.err) || !strings.Contains(err.Error(), "The "+dependency+" has been missing for more than a minute.") ||
			!strings.HasSuffix(err.Error(), tc.check) {
			t.Errorf("%s: %v", dependency, err)
		}
	}
}

// Numbers and bools become strings where the schema has a string, and only there.
func TestCoerce(t *testing.T) {
	var document openAPIDocument
	if err := json.Unmarshal([]byte(`{"components":{"schemas":{
		"cm": {"type":"object","properties":{"data":{"type":"object","additionalProperties":{"type":"string"}},
			"metadata":{"allOf":[{"$ref":"#/components/schemas/meta"}]},"port":{"x-kubernetes-int-or-string":true,"anyOf":[{"type":"integer"},{"type":"string"}]},
			"cpu":{"oneOf":[{"type":"string"},{"type":"number"}]},"old":{"allOf":[{"$ref":"#/components/schemas/io.k8s.apimachinery.pkg.util.intstr.IntOrString"}]}}},
		"io.k8s.apimachinery.pkg.util.intstr.IntOrString": {"type":"string","format":"int-or-string"},
		"meta": {"type":"object","properties":{"labels":{"type":"object","additionalProperties":{"type":"string"}}}}}}}`), &document); err != nil {
		t.Fatal(err)
	}
	value := map[string]any{"data": map[string]any{"PORT": int64(8080), "DEBUG": true, "RATIO": 0.5}, "metadata": map[string]any{"labels": map[string]any{"version": int64(2)}},
		"port": int64(80), "cpu": int64(1), "old": int64(8080)}
	changed := false
	coerce(value, document.Components.Schemas["cm"], document.Components.Schemas, 0, &changed)
	want := map[string]any{"data": map[string]any{"PORT": "8080", "DEBUG": "true", "RATIO": "0.5"}, "metadata": map[string]any{"labels": map[string]any{"version": "2"}},
		"port": int64(80), "cpu": int64(1), "old": int64(8080)}
	if !changed || !reflect.DeepEqual(value, want) {
		t.Fatalf("got %v", value)
	}
}

// The drift comment is a line of its own, and only that text is taken off.
func TestDriftNote(t *testing.T) {
	for _, body := range []string{"kind: A", "kind: A\n", "kind: A\n# Changed outside Terraform: mine\ndata:\n  k: v\n"} {
		note := driftNote(body, []string{"data.a", "spec.b"}, nil)
		if noted := body + note; !strings.Contains(noted, "\n# Changed outside Terraform: data.a, spec.b") || strings.TrimSuffix(noted, note) != body {
			t.Errorf("%q: %q", body, noted)
		}
	}
	if note := driftNote("kind: A\n", []string{"data.a", "data.b"}, map[string]any{"data.a": "2"}); note != "# Changed outside Terraform:\n#   data.a: \"2\"\n#   data.b: (removed)\n" {
		t.Errorf("with values: %q", note)
	}
	// A value with a character YAML reads as a line break stays in the comment.
	body := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: a\n"
	note := driftNote(body, []string{"data.a"}, map[string]any{"data.a": "x\u0085immutable: true"})
	if manifest, err := decodeYAML(body + note); err != nil || manifest["immutable"] != nil || !strings.Contains(note, `\u0085`) {
		t.Errorf("note %q decoded to %v (error %v)", note, manifest, err)
	}
}
