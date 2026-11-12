package kubernetes

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestFlattenPersistentVolumeClaimStatus(t *testing.T) {
	input := corev1.PersistentVolumeClaimStatus{
		Phase: corev1.ClaimBound,
		AccessModes: []corev1.PersistentVolumeAccessMode{
			corev1.ReadWriteOnce,
		},
		Capacity: corev1.ResourceList{
			corev1.ResourceStorage: resource.MustParse("50Gi"),
		},
	}

	output := flattenPersistentVolumeClaimStatus(input)
	if len(output) == 0 {
		t.Fatal("Expected non-empty output")
	}
	m := output[0].(map[string]interface{})

	if m["phase"] != string(corev1.ClaimBound) {
		t.Fatalf("Expected phase=%q, got %q", corev1.ClaimBound, m["phase"])
	}

	accessModes, ok := m["access_modes"].(*schema.Set)
	if !ok {
		t.Fatal("access_modes missing or wrong type")
	}
	if !accessModes.Contains("ReadWriteOnce") {
		t.Fatalf("Expected access_modes to contain ReadWriteOnce, got %#v", accessModes.List())
	}

	capacity, ok := m["capacity"].(map[string]string)
	if !ok {
		t.Fatal("capacity missing or wrong type")
	}
	if capacity["storage"] != "50Gi" {
		t.Fatalf("Expected capacity[storage]=50Gi, got %q", capacity["storage"])
	}
}

func TestFlattenPersistentVolumeClaimStatus_empty(t *testing.T) {
	output := flattenPersistentVolumeClaimStatus(corev1.PersistentVolumeClaimStatus{})
	if len(output) == 0 {
		t.Fatal("Expected non-empty output")
	}
	m := output[0].(map[string]interface{})
	if len(m) != 0 {
		t.Fatalf("Expected empty map, got %#v", m)
	}
}
