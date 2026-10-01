// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWorkloadSelectorComparisonBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, before, after string
		equal               bool
	}{
		{"root-absence", `null`, `[{}]`, false},
		{"root-cardinality", `[]`, `[{}]`, false},
		{"root-null-empty", `null`, `[]`, false},
		{"empty-map", `[{"match_labels":{}}]`, `[{}]`, true},
		{"empty-expressions", `[{"match_expressions":[]}]`, `[{}]`, true},
		{"empty-values", `[{"match_expressions":[{"key":"app","operator":"Exists","values":[]}]}]`, `[{"match_expressions":[{"key":"app","operator":"Exists"}]}]`, true},
		{"empty-string-map-member", `[{"match_labels":{}}]`, `[{"match_labels":{"app":""}}]`, false},
		{"empty-string-set-member", `[{"match_expressions":[{"key":"app","operator":"In","values":[]}]}]`, `[{"match_expressions":[{"key":"app","operator":"In","values":[""]}]}]`, false},
		{"missing-expression", `[{"match_expressions":[]}]`, `[{"match_expressions":[{"key":"app","operator":"Exists"}]}]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := workloadSelectorTestValue(t, tc.before)
			after := workloadSelectorTestValue(t, tc.after)
			for _, values := range [][2]tftypes.Value{{before, after}, {after, before}} {
				original := values[0].Copy()
				equal, err := workloadSelectorsEqual(values[0], values[1])
				if err != nil || equal != tc.equal {
					t.Fatalf("equal = %t, want %t; error: %v", equal, tc.equal, err)
				}
				if !values[0].Equal(original) {
					t.Fatal("comparison mutated its input")
				}
			}
		})
	}
}

func TestWorkloadSelectorUnknownValues(t *testing.T) {
	selector := tftypes.NewAttributePath().WithElementKeyInt(0)
	expression := selector.WithAttributeName("match_expressions").WithElementKeyInt(0)
	for _, at := range []*tftypes.AttributePath{
		tftypes.NewAttributePath(),
		selector,
		selector.WithAttributeName("match_labels"),
		selector.WithAttributeName("match_labels").WithElementKeyString("app"),
		selector.WithAttributeName("match_expressions"),
		expression,
		expression.WithAttributeName("key"),
		expression.WithAttributeName("operator"),
		expression.WithAttributeName("values"),
		expression.WithAttributeName("values").WithElementKeyValue(tftypes.NewValue(tftypes.String, "")),
	} {
		t.Run(at.String(), func(t *testing.T) {
			state := workloadSelectorTestValue(t, `[{"match_labels":{"app":""},"match_expressions":[{"key":"app","operator":"In","values":[""]}]}]`)
			plan, err := tftypes.Transform(state, func(current *tftypes.AttributePath, value tftypes.Value) (tftypes.Value, error) {
				if current.Equal(at) {
					return tftypes.NewValue(value.Type(), tftypes.UnknownValue), nil
				}
				return value, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			original := plan.Copy()
			equal, err := workloadSelectorsEqual(plan, state)
			if err != nil || equal {
				t.Fatalf("unknown difference treated as equal: %t, error: %v", equal, err)
			}
			if !plan.Equal(original) {
				t.Fatal("comparison rewrote an unknown value")
			}
		})
	}
}

func workloadSelectorTestValue(t *testing.T, raw string) tftypes.Value {
	t.Helper()
	typ := tftypes.List{ElementType: labelSelectorBlock(false).NestedObject.Type().TerraformType(context.Background())}
	value, err := (&tfprotov6.RawState{JSON: []byte(raw)}).Unmarshal(typ)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestFlattenWorkloadSelectorOwnership(t *testing.T) {
	emptyMap := types.MapValueMust(types.StringType, map[string]attr.Value{})
	emptySet := types.SetValueMust(types.StringType, []attr.Value{})
	nullMap, nullSet := types.MapNull(types.StringType), types.SetNull(types.StringType)
	requirement := func(key string, values types.Set) LabelSelectorRequirementModel {
		return LabelSelectorRequirementModel{Key: types.StringValue(key), Operator: types.StringValue("Exists"), Values: values}
	}
	apiRequirement := func(key string) metav1.LabelSelectorRequirement {
		return metav1.LabelSelectorRequirement{Key: key, Operator: metav1.LabelSelectorOpExists}
	}
	for _, tc := range []struct {
		name   string
		prior  []LabelSelectorModel
		api    *metav1.LabelSelector
		labels types.Map
		values []types.Set
		empty  bool
		absent bool
	}{
		{"import", nil, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{apiRequirement("app")}}, nullMap, []types.Set{nullSet}, false, false},
		{"preserve-empty", []LabelSelectorModel{{MatchLabels: emptyMap, MatchExpressions: []LabelSelectorRequirementModel{requirement("app", emptySet)}}}, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{apiRequirement("app")}}, emptyMap, []types.Set{emptySet}, false, false},
		{"reordered", []LabelSelectorModel{{MatchLabels: nullMap, MatchExpressions: []LabelSelectorRequirementModel{requirement("app", emptySet), requirement("other", nullSet)}}}, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{apiRequirement("other"), apiRequirement("app")}}, nullMap, []types.Set{nullSet, emptySet}, false, false},
		{"duplicates", []LabelSelectorModel{{MatchLabels: nullMap, MatchExpressions: []LabelSelectorRequirementModel{requirement("app", emptySet), requirement("app", nullSet)}}}, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{apiRequirement("app"), apiRequirement("app")}}, nullMap, []types.Set{emptySet, nullSet}, false, false},
		{"removed-expression", []LabelSelectorModel{{MatchLabels: nullMap, MatchExpressions: []LabelSelectorRequirementModel{requirement("app", emptySet), requirement("other", nullSet)}}}, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{apiRequirement("other")}}, nullMap, []types.Set{nullSet}, false, false},
		{"changed-operator", []LabelSelectorModel{{MatchLabels: nullMap, MatchExpressions: []LabelSelectorRequirementModel{requirement("app", emptySet)}}}, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "app", Operator: metav1.LabelSelectorOpDoesNotExist}}}, nullMap, []types.Set{nullSet}, false, false},
		{"unknown-prior", []LabelSelectorModel{{MatchLabels: types.MapUnknown(types.StringType), MatchExpressions: []LabelSelectorRequirementModel{requirement("app", types.SetUnknown(types.StringType))}}}, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{apiRequirement("app")}}, nullMap, []types.Set{nullSet}, false, false},
		{"api-labels-win", []LabelSelectorModel{{MatchLabels: emptyMap}}, &metav1.LabelSelector{MatchLabels: map[string]string{"app": "database"}}, types.MapValueMust(types.StringType, map[string]attr.Value{"app": types.StringValue("database")}), nil, false, false},
		{"api-values-win", []LabelSelectorModel{{MatchLabels: nullMap, MatchExpressions: []LabelSelectorRequirementModel{requirement("app", emptySet)}}}, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "app", Operator: metav1.LabelSelectorOpIn, Values: []string{"database"}}}}, nullMap, []types.Set{types.SetValueMust(types.StringType, []attr.Value{types.StringValue("database")})}, false, false},
		{"nonempty-prior-removed", []LabelSelectorModel{{MatchLabels: types.MapValueMust(types.StringType, map[string]attr.Value{"app": types.StringValue("")}), MatchExpressions: []LabelSelectorRequirementModel{requirement("app", types.SetValueMust(types.StringType, []attr.Value{types.StringValue("")}))}}}, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{apiRequirement("app")}}, nullMap, []types.Set{nullSet}, false, false},
		{"empty-expression-list", []LabelSelectorModel{{MatchLabels: nullMap, MatchExpressions: []LabelSelectorRequirementModel{}}}, &metav1.LabelSelector{}, nullMap, nil, true, false},
		{"nonempty-expression-list-removed", []LabelSelectorModel{{MatchLabels: nullMap, MatchExpressions: []LabelSelectorRequirementModel{requirement("app", emptySet)}}}, &metav1.LabelSelector{}, nullMap, nil, false, false},
		{"selector-absent", []LabelSelectorModel{{MatchLabels: emptyMap}}, nil, nullMap, nil, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := flattenWorkloadSelector(context.Background(), tc.api, tc.prior)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if tc.absent {
				if got != nil {
					t.Fatal("Read recreated an absent selector")
				}
				return
			}
			if len(got) != 1 || !got[0].MatchLabels.Equal(tc.labels) || len(got[0].MatchExpressions) != len(tc.values) {
				t.Fatalf("unexpected selector: %+v", got)
			}
			if len(tc.values) == 0 && (got[0].MatchExpressions != nil) != tc.empty {
				t.Fatalf("empty expression presence changed: %+v", got[0].MatchExpressions)
			}
			for i, values := range tc.values {
				if !got[0].MatchExpressions[i].Values.Equal(values) {
					t.Errorf("expression %d values = %s, want %s", i, got[0].MatchExpressions[i].Values, values)
				}
			}
		})
	}
}
