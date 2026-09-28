// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildID(t *testing.T) {
	testCases := []struct {
		name string
		meta metav1.ObjectMeta
		want string
	}{
		{"namespaced", metav1.ObjectMeta{Namespace: "team-a", Name: "my-role"}, "team-a/my-role"},
		{"empty namespace still emits the separator", metav1.ObjectMeta{Name: "r"}, "/r"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := BuildID(tc.meta); got != tc.want {
				t.Errorf("BuildID() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseID(t *testing.T) {
	testCases := []struct {
		name          string
		id            string
		wantNamespace string
		wantName      string
		wantErr       string
	}{
		{"namespaced", "team-a/my-role", "team-a", "my-role", ""},
		{"empty namespace", "/my-role", "", "my-role", ""},
		{"empty name", "team-a/", "team-a", "", ""},
		{"bare name is rejected", "my-role", "", "", `Unexpected ID format ("my-role"), expected "namespace/name".`},
		{"three parts are rejected", "a/b/c", "", "", `Unexpected ID format ("a/b/c"), expected "namespace/name".`},
		{"empty string is rejected", "", "", "", `Unexpected ID format (""), expected "namespace/name".`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			namespace, name, err := ParseID(tc.id)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("ParseID(%q) error = %v, want %q", tc.id, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseID(%q) returned %v, want no error", tc.id, err)
			}
			if namespace != tc.wantNamespace || name != tc.wantName {
				t.Errorf("ParseID(%q) = (%q, %q), want (%q, %q)", tc.id, namespace, name, tc.wantNamespace, tc.wantName)
			}
		})
	}
}
