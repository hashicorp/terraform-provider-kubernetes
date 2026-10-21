// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	v1 "k8s.io/api/core/v1"
)

func testStorageClassAllowedTopologies(key string, values ...interface{}) []interface{} {
	return []interface{}{
		map[string]interface{}{
			"match_label_expressions": []interface{}{
				map[string]interface{}{
					"key":    key,
					"values": schema.NewSet(schema.HashString, values),
				},
			},
		},
	}
}

func TestPatchStorageClassAllowedTopologies(t *testing.T) {
	const zone = "topology.ebs.csi.aws.com/zone"

	testCases := []struct {
		name     string
		old      []interface{}
		new      []interface{}
		expected string
	}{
		{
			name:     "values changed",
			old:      testStorageClassAllowedTopologies(zone, "us-east-1a"),
			new:      testStorageClassAllowedTopologies(zone, "us-east-1a", "us-east-1b"),
			expected: `[{"path":"/allowedTopologies","value":[{"matchLabelExpressions":[{"key":"topology.ebs.csi.aws.com/zone","values":["us-east-1a","us-east-1b"]}]}],"op":"add"}]`,
		},
		{
			name:     "added to a storage class without topologies",
			old:      []interface{}{},
			new:      testStorageClassAllowedTopologies(zone, "us-east-1a"),
			expected: `[{"path":"/allowedTopologies","value":[{"matchLabelExpressions":[{"key":"topology.ebs.csi.aws.com/zone","values":["us-east-1a"]}]}],"op":"add"}]`,
		},
		{
			name:     "removed",
			old:      testStorageClassAllowedTopologies(zone, "us-east-1a"),
			new:      []interface{}{},
			expected: `[{"path":"/allowedTopologies","op":"remove"}]`,
		},
		{
			name:     "removed by an empty block",
			old:      testStorageClassAllowedTopologies(zone, "us-east-1a"),
			new:      []interface{}{nil},
			expected: `[{"path":"/allowedTopologies","op":"remove"}]`,
		},
		{
			name:     "unset before and after",
			old:      []interface{}{},
			new:      []interface{}{nil},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ops := patchStorageClassAllowedTopologies(tc.old, tc.new)
			if tc.expected == "" {
				if len(ops) != 0 {
					t.Fatalf("expected no patch operations, got %d", len(ops))
				}
				return
			}
			// values is a schema.Set, so its order follows the set hash.
			for _, op := range ops {
				if add, ok := op.(*AddOperation); ok {
					for _, term := range add.Value.([]v1.TopologySelectorTerm) {
						for _, req := range term.MatchLabelExpressions {
							sort.Strings(req.Values)
						}
					}
				}
			}
			data, err := ops.MarshalJSON()
			if err != nil {
				t.Fatalf("failed to marshal patch operations: %s", err)
			}
			if string(data) != tc.expected {
				t.Fatalf("unexpected patch:\n got: %s\nwant: %s", data, tc.expected)
			}
		})
	}
}
