// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestIsInternalKey(t *testing.T) {
	t.Parallel()

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
		t.Run(tc.Key, func(t *testing.T) {
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
	uid := "7e9439cb-2584-4b50-81bc-441127e11b26"
	cases := map[string]struct {
		meta     metav1.ObjectMeta
		expected []interface{}
	}{
		"DefaultNamespaceStaticName": {
			metav1.ObjectMeta{
				Annotations:     annotations,
				GenerateName:    "",
				Generation:      1,
				Labels:          labels,
				Name:            "foo",
				Namespace:       "",
				ResourceVersion: "1",
				UID:             types.UID(uid),
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
		"NonDefaultNamespaceStaticName": {
			metav1.ObjectMeta{
				Annotations:     annotations,
				GenerateName:    "",
				Generation:      1,
				Labels:          labels,
				Name:            "foo",
				Namespace:       "Test",
				ResourceVersion: "1",
				UID:             types.UID(uid),
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
		"DefaultNamespaceGeneratedName": {
			metav1.ObjectMeta{
				Annotations:     annotations,
				GenerateName:    "gen-foo",
				Generation:      1,
				Labels:          labels,
				Name:            "",
				Namespace:       "",
				ResourceVersion: "1",
				UID:             types.UID(uid),
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
		"NonDefaultNamespaceGeneratedName": {
			metav1.ObjectMeta{
				Annotations:     annotations,
				GenerateName:    "gen-foo",
				Generation:      1,
				Labels:          labels,
				Name:            "",
				Namespace:       "Test",
				ResourceVersion: "1",
				UID:             types.UID(uid),
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
	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			out := flattenMetadataFields(c.meta)
			if !reflect.DeepEqual(out, c.expected) {
				t.Fatalf("Error matching output and expected: %#v vs %#v", out, c.expected)
			}
		})
	}
}

// TestFlattenMetadata aims to validate whether or not 'ignore_annotations' and 'ignore_labels'
// are cut out along with well-known Kubernetes annotations and labels.
func TestFlattenMetadata(t *testing.T) {
	t.Parallel()

	uid := "7e9439cb-2584-4b50-81bc-441127e11b26"
	cases := map[string]struct {
		meta         metav1.ObjectMeta
		providerMeta kubeClientsets
		expected     []interface{}
	}{
		"IgnoreAnnotations": {
			metav1.ObjectMeta{
				Annotations: map[string]string{
					"fake.kubernetes.io": "fake",
					"foo.example.com":    "bar",
					"bar.example.com":    "foo",
				},
				GenerateName: "",
				Generation:   1,
				Labels: map[string]string{
					"foo": "bar",
					"bar": "foo",
				},
				Name:            "foo",
				Namespace:       "",
				ResourceVersion: "1",
				UID:             types.UID(uid),
			},
			kubeClientsets{
				IgnoreAnnotations: []string{"foo.example.com"},
				IgnoreLabels:      []string{},
			},
			[]interface{}{map[string]interface{}{
				"annotations": map[string]string{
					"bar.example.com": "foo",
				},
				"generation": int64(1),
				"labels": map[string]string{
					"foo": "bar",
					"bar": "foo",
				},
				"name":             "foo",
				"resource_version": "1",
				"uid":              uid,
			}},
		},
		"IgnoreLabels": {
			metav1.ObjectMeta{
				Annotations: map[string]string{
					"fake.kubernetes.io": "fake",
					"foo.example.com":    "bar",
					"bar.example.com":    "foo",
				},
				GenerateName: "",
				Generation:   1,
				Labels: map[string]string{
					"foo": "bar",
					"bar": "foo",
				},
				Name:            "foo",
				Namespace:       "",
				ResourceVersion: "1",
				UID:             types.UID(uid),
			},
			kubeClientsets{
				IgnoreAnnotations: []string{},
				IgnoreLabels:      []string{"foo"},
			},
			[]interface{}{map[string]interface{}{
				"annotations": map[string]string{
					"foo.example.com": "bar",
					"bar.example.com": "foo",
				},
				"generation": int64(1),
				"labels": map[string]string{
					"bar": "foo",
				},
				"name":             "foo",
				"resource_version": "1",
				"uid":              uid,
			}},
		},
		"IgnoreAnnotationsAndLabels": {
			metav1.ObjectMeta{
				Annotations: map[string]string{
					"fake.kubernetes.io": "fake",
					"foo.example.com":    "bar",
					"bar.example.com":    "foo",
				},
				GenerateName: "",
				Generation:   1,
				Labels: map[string]string{
					"foo": "bar",
					"bar": "foo",
				},
				Name:            "foo",
				Namespace:       "",
				ResourceVersion: "1",
				UID:             types.UID(uid),
			},
			kubeClientsets{
				IgnoreAnnotations: []string{"foo.example.com"},
				IgnoreLabels:      []string{"foo"},
			},
			[]interface{}{map[string]interface{}{
				"annotations": map[string]string{
					"bar.example.com": "foo",
				},
				"generation": int64(1),
				"labels": map[string]string{
					"bar": "foo",
				},
				"name":             "foo",
				"resource_version": "1",
				"uid":              uid,
			}},
		},
	}
	rawData := map[string]interface{}{
		"metadata": []interface{}{map[string]interface{}{
			"annotations":      map[string]interface{}{},
			"generation":       0,
			"labels":           map[string]interface{}{},
			"name":             "",
			"resource_version": "",
			"uid":              "",
		}},
	}
	d := schema.TestResourceDataRaw(t, map[string]*schema.Schema{"metadata": namespacedMetadataSchema("fake", true)}, rawData)
	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			out := flattenMetadata(c.meta, d, c.providerMeta)
			if !reflect.DeepEqual(out, c.expected) {
				t.Fatalf("Error matching output and expected: %#v vs %#v", out, c.expected)
			}
		})
	}
}
