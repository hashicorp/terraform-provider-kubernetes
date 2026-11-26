// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
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
		"namespace missing":            {apierrors.NewNotFound(schema.GroupResource{Resource: "namespaces"}, "prod"), true},
		"resource not served yet":      {&apierrors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound, Code: 404}}, true},
		"the object itself is missing": {apierrors.NewNotFound(configMaps, "settings"), false},
		"conflict":                     {apierrors.NewConflict(configMaps, "settings", nil), false},
		"rejected":                     {apierrors.NewBadRequest("unknown field"), false},
	} {
		if got := awaitsDependency(tc.err); got != tc.want {
			t.Errorf("%s: got %t, want %t", name, got, tc.want)
		}
	}
}
