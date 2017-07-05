package kubernetes

import (
	"testing"
)

func TestValidateModeBits(t *testing.T) {
	validCases := []int{
		0, 0001, 0644, 0777,
	}
	for _, mode := range validCases {
		_, es := validateModeBits(mode, "mode")
		if len(es) > 0 {
			t.Fatalf("Expected %#o to be valid: %#v", mode, es)
		}
	}

	invalidCases := []int{
		-5, -1, 512, 777,
	}
	for _, mode := range invalidCases {
		_, es := validateModeBits(mode, "mode")
		if len(es) == 0 {
			t.Fatalf("Expected %#o to be invalid", mode)
		}
	}
}
