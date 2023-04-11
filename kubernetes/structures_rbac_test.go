// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"reflect"
	"testing"

	api "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExpandClusterRoleAggregationRule(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *api.AggregationRule
	}{
		{
			[]interface{}{},
			&api.AggregationRule{},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"cluster_role_selectors": []interface{}{
						map[string]interface{}{
							"match_labels": map[string]interface{}{"key": "value"},
						},
					},
				},
			},
			&api.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{
						MatchLabels: map[string]string{"key": "value"},
					},
				},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"cluster_role_selectors": []interface{}{
						map[string]interface{}{
							"match_labels": map[string]interface{}{"key": "value"},
						},
						map[string]interface{}{
							"match_labels": map[string]interface{}{"foo": "bar"},
						},
					},
				},
			},
			&api.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{
						MatchLabels: map[string]string{"key": "value"},
					},
					{
						MatchLabels: map[string]string{"foo": "bar"},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		output := expandClusterRoleAggregationRule(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from expander.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestFlattenClusterRoleAggregationRule(t *testing.T) {
	cases := []struct {
		Input          *api.AggregationRule
		ExpectedOutput []interface{}
	}{
		{
			&api.AggregationRule{},
			[]interface{}{map[string]interface{}{}},
		},
		{
			&api.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{
						MatchLabels: map[string]string{"key": "value"},
					},
				},
			},
			[]interface{}{
				map[string]interface{}{
					"cluster_role_selectors": []interface{}{
						map[string]interface{}{
							"match_labels": map[string]string{"key": "value"},
						},
					},
				},
			},
		},
		{
			&api.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{
						MatchLabels: map[string]string{"key": "value"},
					},
					{
						MatchLabels: map[string]string{"foo": "bar"},
					},
				},
			},
			[]interface{}{
				map[string]interface{}{
					"cluster_role_selectors": []interface{}{
						map[string]interface{}{
							"match_labels": map[string]string{"key": "value"},
						},
						map[string]interface{}{
							"match_labels": map[string]string{"foo": "bar"},
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		output := flattenClusterRoleAggregationRule(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}
