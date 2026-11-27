// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Only a write refused for the object's revision is tried again.
func TestStale(t *testing.T) {
	configMaps := schema.GroupResource{Resource: "configmaps"}
	owned := apierrors.NewConflict(configMaps, "a", nil)
	owned.ErrStatus.Details.Causes = []metav1.StatusCause{{Type: metav1.CauseTypeFieldManagerConflict, Field: ".data.a"}}
	for name, tc := range map[string]struct {
		err  error
		want bool
	}{
		"changed since it was read": {apierrors.NewConflict(configMaps, "a", errors.New("the object has been modified")), true},
		"field ownership conflict":  {owned, false},
		"object gone":               {apierrors.NewConflict(configMaps, "a", errors.New("uid mismatch: no existing object was found")), false},
		"not found":                 {apierrors.NewNotFound(configMaps, "a"), false},
		"no error":                  {nil, false},
	} {
		if got := stale(tc.err); got != tc.want {
			t.Errorf("%s: got %t, want %t", name, got, tc.want)
		}
	}
}

// A write refused because the object is gone is told by the status of the
// error, not by its text, which holds the name of the object.
func TestGone(t *testing.T) {
	configMaps := schema.GroupResource{Resource: "configmaps"}
	other := apierrors.NewInvalid(schema.GroupKind{Kind: "ConfigMap"}, "a", nil)
	other.ErrStatus.Details.Causes = []metav1.StatusCause{{Type: metav1.CauseTypeFieldValueInvalid, Field: "metadata.uid"}}
	for name, tc := range map[string]struct {
		err               error
		deleted, replaced bool
	}{
		"apply to a missing object": {apierrors.NewConflict(configMaps, "a", errors.New("uid mismatch: no existing object was found")), true, false},
		"not found":                 {apierrors.NewNotFound(configMaps, "a"), true, false},
		"another UID":               {other, false, true},
		"read again as another":     {fmt.Errorf("%w: its UID is b, not a", errReplaced), false, true},
		"forbidden, name like UID":  {apierrors.NewForbidden(configMaps, "metadata.uid-mismatch", errors.New("cannot patch")), false, false},
		"changed since it was read": {apierrors.NewConflict(configMaps, "a", errors.New("the object has been modified")), false, false},
	} {
		if deleted, replaced := gone(tc.err); deleted != tc.deleted || replaced != tc.replaced {
			t.Errorf("%s: got deleted %t, replaced %t", name, deleted, replaced)
		}
	}
}

// Fields applied under the manager that no manifest of this resource set are
// another resource's.
func TestForeignFields(t *testing.T) {
	live := &unstructured.Unstructured{Object: map[string]any{}}
	live.SetManagedFields([]metav1.ManagedFieldsEntry{
		entry("terraform-patch", "Apply", `{"f:data":{"f:a":{},"f:b":{}},"f:spec":{"f:containers":{"k:{\"name\":\"web\"}":{".":{},"f:name":{},
			"f:env":{"k:{\"name\":\"A\"}":{".":{},"f:name":{},"f:value":{}}}}}}}`),
		entry("owner", "Update", `{"f:data":{"f:c":{}}}`),
	})
	manifest := decodeJSON(t, `{"data":{"a":"1"},"spec":{"containers":[{"name":"web","env":[{"name":"A","value":"1"}]}]}}`)
	for name, tc := range map[string]struct {
		manifests []map[string]any
		want      []string
	}{
		"another resource's":    {[]map[string]any{manifest}, []string{".data.b"}},
		"set by the last apply": {[]map[string]any{decodeJSON(t, `{"data":{"b":"1"}}`), manifest}, nil},
		"none set": {[]map[string]any{{}}, []string{".data.a", ".data.b", ".spec.containers[name=web].env[name=A].name",
			".spec.containers[name=web].env[name=A].value", ".spec.containers[name=web].name"}},
	} {
		t.Run(name, func(t *testing.T) {
			if got := fieldList(foreignFields(live, "terraform-patch", tc.manifests...), ""); !slices.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
