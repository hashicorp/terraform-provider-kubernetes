package kubernetes

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func TestFlattenStatefulSetSpecUpdateStrategy_maxUnavailable(t *testing.T) {
	maxUnavailable := intstr.FromInt32(2)
	input := v1.StatefulSetUpdateStrategy{
		Type: v1.RollingUpdateStatefulSetStrategyType,
		RollingUpdate: &v1.RollingUpdateStatefulSetStrategy{
			Partition:      ptr.To(int32(1)),
			MaxUnavailable: &maxUnavailable,
		},
	}

	output := flattenStatefulSetSpecUpdateStrategy(input)
	if len(output) == 0 {
		t.Fatal("Expected non-empty output")
	}
	m := output[0].(map[string]interface{})
	ru := m["rolling_update"].([]interface{})
	if len(ru) == 0 {
		t.Fatal("Expected rolling_update to be non-empty")
	}
	ruMap := ru[0].(map[string]interface{})

	got, ok := ruMap["max_unavailable"]
	if !ok {
		t.Fatal("max_unavailable missing from flattened output")
	}
	if got != "2" {
		t.Fatalf("Expected max_unavailable='2', got %q", got)
	}
	if ruMap["partition"] != int32(1) {
		t.Fatalf("Expected partition=1, got %v", ruMap["partition"])
	}
}

func TestFlattenStatefulSetSpecUpdateStrategy_noMaxUnavailable(t *testing.T) {
	input := v1.StatefulSetUpdateStrategy{
		Type: v1.RollingUpdateStatefulSetStrategyType,
		RollingUpdate: &v1.RollingUpdateStatefulSetStrategy{
			Partition: ptr.To(int32(0)),
		},
	}

	output := flattenStatefulSetSpecUpdateStrategy(input)
	m := output[0].(map[string]interface{})
	ru := m["rolling_update"].([]interface{})
	ruMap := ru[0].(map[string]interface{})

	if _, ok := ruMap["max_unavailable"]; ok {
		t.Fatal("max_unavailable should not be present when nil")
	}
}

func TestExpandStatefulSetSpecUpdateStrategy_maxUnavailable(t *testing.T) {
	maxUnavailable := intstr.FromString("25%")
	input := []interface{}{
		map[string]interface{}{
			"type": string(v1.RollingUpdateStatefulSetStrategyType),
			"rolling_update": []interface{}{
				map[string]interface{}{
					"partition":       0,
					"max_unavailable": "25%",
				},
			},
		},
	}

	output, err := expandStatefulSetSpecUpdateStrategy(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if output.RollingUpdate == nil {
		t.Fatal("Expected RollingUpdate to be set")
	}
	if output.RollingUpdate.MaxUnavailable == nil {
		t.Fatal("Expected MaxUnavailable to be set")
	}
	if !reflect.DeepEqual(*output.RollingUpdate.MaxUnavailable, maxUnavailable) {
		t.Fatalf("Expected MaxUnavailable=%#v, got %#v", maxUnavailable, *output.RollingUpdate.MaxUnavailable)
	}
}

func TestExpandStatefulSetSpecUpdateStrategy_noMaxUnavailable(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"type": string(v1.RollingUpdateStatefulSetStrategyType),
			"rolling_update": []interface{}{
				map[string]interface{}{
					"partition": 0,
				},
			},
		},
	}

	output, err := expandStatefulSetSpecUpdateStrategy(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if output.RollingUpdate == nil {
		t.Fatal("Expected RollingUpdate to be set")
	}
	if output.RollingUpdate.MaxUnavailable != nil {
		t.Fatal("Expected MaxUnavailable to be nil when not set")
	}
}
