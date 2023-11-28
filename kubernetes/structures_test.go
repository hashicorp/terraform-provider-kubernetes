// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestIsInternalKey(t *testing.T) {
	testCases := []struct {
		Key      string
		Expected bool
	}{
		{"", false},
		{"anyKey", false},
		{"any.hostname.io", false},
		{"any.hostname.com/with/path", false},
		{"service.beta.kubernetes.io/aws-load-balancer-backend-protocol", false},
		{"app.kubernetes.io", false},
		{"kubernetes.io", true},
		{"kubectl.kubernetes.io", true},
		{"pv.kubernetes.io/any/path", true},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s", tc.Key), func(t *testing.T) {
			isInternal := isInternalKey(tc.Key)
			if tc.Expected && isInternal != tc.Expected {
				t.Fatalf("Expected %q to be internal", tc.Key)
			}
			if !tc.Expected && isInternal != tc.Expected {
				t.Fatalf("Expected %q not to be internal", tc.Key)
			}
		})
	}
}

func TestFlattenMetadataFields(t *testing.T) {
	t.Parallel()

	annotations := map[string]string{
		"fake.kubernetes.io": "fake",
	}
	labels := map[string]string{
		"foo": "bar",
	}
	uid := types.UID("7e9439cb-2584-4b50-81bc-441127e11b26")
	cases := []struct {
		meta     metav1.ObjectMeta
		expected []interface{}
	}{
		{
			// Default namespace and static name
			metav1.ObjectMeta{
				Annotations:     annotations,
				GenerateName:    "",
				Generation:      1,
				Labels:          labels,
				Name:            "foo",
				Namespace:       "",
				ResourceVersion: "1",
				UID:             uid,
			},
			[]interface{}{map[string]interface{}{
				"annotations":      annotations,
				"generation":       int64(1),
				"labels":           labels,
				"name":             "foo",
				"resource_version": "1",
				"uid":              uid,
			}},
		},
		{
			// Non default namespace and static name,
			metav1.ObjectMeta{
				Annotations:     annotations,
				GenerateName:    "",
				Generation:      1,
				Labels:          labels,
				Name:            "foo",
				Namespace:       "Test",
				ResourceVersion: "1",
				UID:             uid,
			},
			[]interface{}{map[string]interface{}{
				"annotations":      annotations,
				"generation":       int64(1),
				"labels":           labels,
				"name":             "foo",
				"namespace":        "Test",
				"resource_version": "1",
				"uid":              uid,
			}},
		},
		{
			// Default namespace and generated name
			metav1.ObjectMeta{
				Annotations:     annotations,
				GenerateName:    "gen-foo",
				Generation:      1,
				Labels:          labels,
				Name:            "",
				Namespace:       "",
				ResourceVersion: "1",
				UID:             uid,
			},
			[]interface{}{map[string]interface{}{
				"annotations":      annotations,
				"generate_name":    "gen-foo",
				"generation":       int64(1),
				"labels":           labels,
				"name":             "",
				"resource_version": "1",
				"uid":              uid,
			}},
		},
		{
			// Non default namespace and generated name
			metav1.ObjectMeta{
				Annotations:     annotations,
				GenerateName:    "gen-foo",
				Generation:      1,
				Labels:          labels,
				Name:            "",
				Namespace:       "Test",
				ResourceVersion: "1",
				UID:             uid,
			},
			[]interface{}{map[string]interface{}{
				"annotations":      annotations,
				"generate_name":    "gen-foo",
				"generation":       int64(1),
				"labels":           labels,
				"name":             "",
				"namespace":        "Test",
				"resource_version": "1",
				"uid":              uid,
			}},
		},
	}
	for _, c := range cases {
		out := flattenMetadataFields(c.meta)
		if !reflect.DeepEqual(out, c.expected) {
			t.Fatalf("Error matching output and expected: %#v vs %#v", out, c.expected)
		}
	}
}
