package kubernetes

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFlattenPodAffinityTerms_matchLabelKeys(t *testing.T) {
	input := []v1.PodAffinityTerm{
		{
			TopologyKey:       "topology.kubernetes.io/zone",
			MatchLabelKeys:    []string{"pod-template-hash"},
			MismatchLabelKeys: []string{"app"},
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
		},
	}

	output := flattenPodAffinityTerms(input)
	if len(output) != 1 {
		t.Fatalf("Expected 1 element, got %d", len(output))
	}
	m := output[0].(map[string]interface{})

	if v, ok := m["match_label_keys"].(*schema.Set); ok {
		if v.Len() != 1 || !v.Contains("pod-template-hash") {
			t.Fatalf("Expected match_label_keys to contain 'pod-template-hash', got %#v", v.List())
		}
	} else {
		t.Fatal("match_label_keys missing or wrong type")
	}

	if v, ok := m["mismatch_label_keys"].(*schema.Set); ok {
		if v.Len() != 1 || !v.Contains("app") {
			t.Fatalf("Expected mismatch_label_keys to contain 'app', got %#v", v.List())
		}
	} else {
		t.Fatal("mismatch_label_keys missing or wrong type")
	}
}

func TestFlattenPodAffinityTerms_noMatchLabelKeys(t *testing.T) {
	input := []v1.PodAffinityTerm{
		{
			TopologyKey: "topology.kubernetes.io/zone",
		},
	}

	output := flattenPodAffinityTerms(input)
	if len(output) != 1 {
		t.Fatalf("Expected 1 element, got %d", len(output))
	}
	m := output[0].(map[string]interface{})

	if _, ok := m["match_label_keys"]; ok {
		t.Fatal("match_label_keys should not be present when empty")
	}
	if _, ok := m["mismatch_label_keys"]; ok {
		t.Fatal("mismatch_label_keys should not be present when empty")
	}
}

func TestExpandPodAffinityTerms_matchLabelKeys(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"topology_key":        "topology.kubernetes.io/zone",
			"match_label_keys":    schema.NewSet(schema.HashString, []interface{}{"pod-template-hash"}),
			"mismatch_label_keys": schema.NewSet(schema.HashString, []interface{}{"app"}),
		},
	}

	output := expandPodAffinityTerms(input)
	if len(output) != 1 {
		t.Fatalf("Expected 1 element, got %d", len(output))
	}
	term := output[0]

	if len(term.MatchLabelKeys) != 1 || term.MatchLabelKeys[0] != "pod-template-hash" {
		t.Fatalf("Expected MatchLabelKeys=[pod-template-hash], got %#v", term.MatchLabelKeys)
	}
	if len(term.MismatchLabelKeys) != 1 || term.MismatchLabelKeys[0] != "app" {
		t.Fatalf("Expected MismatchLabelKeys=[app], got %#v", term.MismatchLabelKeys)
	}
}

func TestExpandPodAffinityTerms_noMatchLabelKeys(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"topology_key": "topology.kubernetes.io/zone",
		},
	}

	output := expandPodAffinityTerms(input)
	if len(output) != 1 {
		t.Fatalf("Expected 1 element, got %d", len(output))
	}
	term := output[0]

	if len(term.MatchLabelKeys) != 0 {
		t.Fatalf("Expected empty MatchLabelKeys, got %#v", term.MatchLabelKeys)
	}
	if len(term.MismatchLabelKeys) != 0 {
		t.Fatalf("Expected empty MismatchLabelKeys, got %#v", term.MismatchLabelKeys)
	}
}
