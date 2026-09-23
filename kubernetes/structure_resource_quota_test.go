package kubernetes

import (
	"testing"
)

func TestExpandResourceQuotaSpec_emptyHard(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"hard": map[string]interface{}{},
		},
	}

	output, err := expandResourceQuotaSpec(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if output == nil {
		t.Fatal("Expected non-nil output")
	}
}

func TestExpandResourceQuotaSpec_nilHard(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"hard": nil,
		},
	}

	output, err := expandResourceQuotaSpec(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if output == nil {
		t.Fatal("Expected non-nil output")
	}
}

func TestExpandResourceQuotaSpec_noHard(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{},
	}

	output, err := expandResourceQuotaSpec(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if output == nil {
		t.Fatal("Expected non-nil output")
	}
}

func TestExpandResourceQuotaSpec_emptyInput(t *testing.T) {
	output, err := expandResourceQuotaSpec([]interface{}{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if output == nil {
		t.Fatal("Expected non-nil output")
	}
}
