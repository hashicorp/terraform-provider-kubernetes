// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func testAllowedTopologies(key string, values ...string) []AllowedTopologyModel {
	elems := make([]attr.Value, len(values))
	for i, v := range values {
		elems[i] = types.StringValue(v)
	}
	return []AllowedTopologyModel{{
		MatchLabelExpressions: []MatchLabelExpressionModel{{
			Key:    types.StringValue(key),
			Values: types.SetValueMust(types.StringType, elems),
		}},
	}}
}

func TestPatchAllowedTopologies(t *testing.T) {
	const zone = "topology.ebs.csi.aws.com/zone"

	testCases := []struct {
		name     string
		state    []AllowedTopologyModel
		plan     []AllowedTopologyModel
		expected string
	}{
		{
			name:     "values changed",
			state:    testAllowedTopologies(zone, "us-east-1a"),
			plan:     testAllowedTopologies(zone, "us-east-1a", "us-east-1b"),
			expected: `[{"path":"/allowedTopologies","value":[{"matchLabelExpressions":[{"key":"topology.ebs.csi.aws.com/zone","values":["us-east-1a","us-east-1b"]}]}],"op":"add"}]`,
		},
		{
			name:     "added to a storage class without topologies",
			state:    nil,
			plan:     testAllowedTopologies(zone, "us-east-1a"),
			expected: `[{"path":"/allowedTopologies","value":[{"matchLabelExpressions":[{"key":"topology.ebs.csi.aws.com/zone","values":["us-east-1a"]}]}],"op":"add"}]`,
		},
		{
			name:     "removed",
			state:    testAllowedTopologies(zone, "us-east-1a"),
			plan:     nil,
			expected: `[{"path":"/allowedTopologies","value":[],"op":"add"}]`,
		},
		{
			name:     "unchanged",
			state:    testAllowedTopologies(zone, "us-east-1a"),
			plan:     testAllowedTopologies(zone, "us-east-1a"),
			expected: "",
		},
		{
			name:     "unset before and after",
			state:    nil,
			plan:     nil,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ops := patchAllowedTopologies(context.Background(), tc.state, tc.plan)
			if tc.expected == "" {
				if len(ops) != 0 {
					t.Fatalf("expected no patch operations, got %d", len(ops))
				}
				return
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
