package kubernetes

import (
	"fmt"
	"testing"
)

func TestDiffStringMap(t *testing.T) {
	testCases := []struct {
		Path        string
		Old         map[string]interface{}
		New         map[string]interface{}
		ExpectedOps PatchOperations
	}{
		{
			Path: "/parent/",
			Old: map[string]interface{}{
				"one": "111",
				"two": "222",
			},
			New: map[string]interface{}{
				"one":   "111",
				"two":   "222",
				"three": "333",
			},
			ExpectedOps: []PatchOperation{
				&AddOperation{
					Path:  "/parent/three",
					Value: "333",
				},
			},
		},
		{
			Path: "/parent/",
			Old: map[string]interface{}{
				"one": "111",
				"two": "222",
			},
			New: map[string]interface{}{
				"one": "111",
				"two": "abcd",
			},
			ExpectedOps: []PatchOperation{
				&ReplaceOperation{
					Path:  "/parent/two",
					Value: "abcd",
				},
			},
		},
		{
			Path: "/parent/",
			Old: map[string]interface{}{
				"one": "111",
				"two": "222",
			},
			New: map[string]interface{}{
				"two":   "abcd",
				"three": "333",
			},
			ExpectedOps: []PatchOperation{
				&RemoveOperation{Path: "/parent/one"},
				&ReplaceOperation{
					Path:  "/parent/two",
					Value: "abcd",
				},
				&AddOperation{
					Path:  "/parent/three",
					Value: "333",
				},
			},
		},
		{
			Path: "/parent/",
			Old: map[string]interface{}{
				"one": "111",
				"two": "222",
			},
			New: map[string]interface{}{
				"two": "222",
			},
			ExpectedOps: []PatchOperation{
				&RemoveOperation{Path: "/parent/one"},
			},
		},
		{
			Path: "/parent/",
			Old: map[string]interface{}{
				"one": "111",
				"two": "222",
			},
			New: map[string]interface{}{},
			ExpectedOps: []PatchOperation{
				&RemoveOperation{Path: "/parent/one"},
				&RemoveOperation{Path: "/parent/two"},
			},
		},
		{
			Path: "/parent/",
			Old:  map[string]interface{}{},
			New: map[string]interface{}{
				"one": "111",
				"two": "222",
			},
			ExpectedOps: []PatchOperation{
				&AddOperation{
					Path: "/parent",
					Value: map[string]interface{}{
						"one": "111",
						"two": "222",
					},
				},
			},
		},
		{
			Path: "/parent/",
			Old: map[string]interface{}{
				"two~with-tilde":           "220",
				"three/with/three/slashes": "330",
			},
			New: map[string]interface{}{
				"one/with-slash":           "111",
				"three/with/three/slashes": "333",
			},
			ExpectedOps: []PatchOperation{
				&AddOperation{
					Path:  "/parent/one~1with-slash",
					Value: "111",
				},
				&RemoveOperation{
					Path: "/parent/two~0with-tilde",
				},
				&ReplaceOperation{
					Path:  "/parent/three~1with~1three~1slashes",
					Value: "333",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			ops := diffStringMap(tc.Path, tc.Old, tc.New)
			if !tc.ExpectedOps.Equal(ops) {
				t.Fatalf("Operations don't match.\nExpected: %v\nGiven:    %v\n", tc.ExpectedOps, ops)
			}
		})
	}
}

func TestEscapeJsonPointer(t *testing.T) {
	testCases := []struct {
		Input          string
		ExpectedOutput string
	}{
		{"simple", "simple"},
		{"special-chars,but no escaping", "special-chars,but no escaping"},
		{"escape-this/forward-slash", "escape-this~1forward-slash"},
		{"escape-this~tilde", "escape-this~0tilde"},
	}
	for _, tc := range testCases {
		output := escapeJsonPointer(tc.Input)
		if output != tc.ExpectedOutput {
			t.Fatalf("Expected %q as after escaping %q, given: %q",
				tc.ExpectedOutput, tc.Input, output)
		}
	}
}
