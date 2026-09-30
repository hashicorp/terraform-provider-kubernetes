// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestJobComparisonMatchesContainerNames(t *testing.T) {
	containerType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":              tftypes.String,
		"image_pull_policy": tftypes.String,
		"resources":         tftypes.Map{ElementType: tftypes.String},
	}}
	listType := tftypes.List{ElementType: containerType}
	container := func(name string, policy, resources any) tftypes.Value {
		return tftypes.NewValue(containerType, map[string]tftypes.Value{
			"name":              tftypes.NewValue(tftypes.String, name),
			"image_pull_policy": tftypes.NewValue(tftypes.String, policy),
			"resources":         tftypes.NewValue(containerType.AttributeTypes["resources"], resources),
		})
	}
	previous := map[string]tftypes.Value{
		"a": container("a", "Always", map[string]tftypes.Value{"cpu": tftypes.NewValue(tftypes.String, "100m")}),
		"b": container("b", "IfNotPresent", map[string]tftypes.Value{"cpu": tftypes.NewValue(tftypes.String, "200m")}),
	}
	for _, name := range []string{"container", "init_container"} {
		t.Run(name, func(t *testing.T) {
			rootType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{name: listType}}
			root := func(values []tftypes.Value) tftypes.Value {
				return tftypes.NewValue(rootType, map[string]tftypes.Value{
					name: tftypes.NewValue(listType, values),
				})
			}
			field := valueField{children: map[string]valueField{
				name: {children: map[string]valueField{
					"name":              {},
					"image_pull_policy": {computed: true},
					"resources":         {computed: true},
				}},
			}}
			for _, test := range []struct {
				name  string
				names []string
			}{
				{"remove first", []string{"b"}},
				{"reorder", []string{"b", "a"}},
				{"new name", []string{"c"}},
				{"retain and add", []string{"b", "c"}},
			} {
				t.Run(test.name, func(t *testing.T) {
					var planned, configured []tftypes.Value
					for _, name := range test.names {
						planned = append(planned, container(name, tftypes.UnknownValue, tftypes.UnknownValue))
						configured = append(configured, container(name, nil, nil))
					}
					plan := root(planned)
					result, ok := jobComparisonValue(field, plan, root([]tftypes.Value{previous["a"], previous["b"]}), root(configured))
					if !ok {
						t.Fatal("computed comparison values could not be resolved")
					}
					var fields map[string]tftypes.Value
					if err := result.As(&fields); err != nil {
						t.Fatal(err)
					}
					var actual []tftypes.Value
					if err := fields[name].As(&actual); err != nil {
						t.Fatal(err)
					}
					for index, containerName := range test.names {
						expected, exists := previous[containerName]
						if !exists {
							expected = container(containerName, nil, nil)
						}
						if !actual[index].Equal(expected) {
							t.Fatalf("container %q inherited wrong computed values:\nactual: %s\nexpected: %s",
								containerName, actual[index], expected)
						}
					}
					if plan.IsFullyKnown() {
						t.Fatal("comparison resolution changed the actual plan")
					}
				})
			}
		})
	}
}
