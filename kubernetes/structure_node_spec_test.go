package kubernetes

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
)

// Test Flatteners
func TestFlattenTaints(t *testing.T) {
	in := []v1.Taint{
		{
			Key:    "node-role.kubernetes.io/spot-worker",
			Value:  "true",
			Effect: "NoExecute",
		},
		{
			Key:    "node-role.kubernetes.io/spot-master",
			Value:  "true",
			Effect: "PreferNoSchedule",
		},
	}
	out := []interface{}{
		map[string]interface{}{
			"key":    "node-role.kubernetes.io/spot-worker",
			"value":  "true",
			"effect": "NoExecute",
		},
		map[string]interface{}{
			"key":    "node-role.kubernetes.io/spot-master",
			"value":  "true",
			"effect": "PreferNoSchedule",
		},
	}

	flatTaints := flattenTaints(in)

	if len(flatTaints) < len(out) {
		t.Error("Failed to flatten taints")
	}

	for i, v := range flatTaints {
		control := v.(map[string]interface{})
		sample := out[i]

		if !reflect.DeepEqual(control, sample) {
			t.Errorf("Unexpected result:\n\tWant:%s\n\tGot:%s\n", control, sample)
		}
	}
}
