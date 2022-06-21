package validators

import (
	"testing"
)

func TestValidateModeBits(t *testing.T) {
	validCases := []string{
		"0", "0001", "0644", "0777",
	}
	for _, mode := range validCases {
		_, es := ValidateModeBits(mode, "mode")
		if len(es) > 0 {
			t.Fatalf("Expected %s to be valid: %#v", mode, es)
		}
	}

	invalidCases := []string{
		"-5", "-1", "512", "777",
	}
	for _, mode := range invalidCases {
		_, es := ValidateModeBits(mode, "mode")
		if len(es) == 0 {
			t.Fatalf("Expected %s to be invalid", mode)
		}
	}
}

func TestValidateBase64Encoded(t *testing.T) {
	validCases := []interface{}{
		"",         // the plain empty string
		"Cg==",     // the encoded empty string
		"blah",     // "nV"
		"VGVzdAo=", // "Test"
		"f0VMRgIBAQAAAAAAAAAAAAMAPgABAAAAMGEAAAAAAABAAAAAAAAAAKATAgAAAAAAAAAAAEAAOAALAEAAHAAbAAYAAAAEAAAAQAAAAAAAAABAAAAAAAAAAEAAAAAAAAAAaAIAAA==", // `head /bin/ls -c 100 | base64`
	}
	for _, data := range validCases {
		_, es := ValidateBase64Encoded(data, "binary_data")
		if len(es) > 0 {
			t.Fatalf("Expected %#o to be valid: %#v", data, es)
		}
	}

	invalidCases := []interface{}{
		nil,
		"bl ah",
		"blahd",
		"C=",
	}
	for _, data := range invalidCases {
		_, es := ValidateBase64Encoded(data, "binary_data")
		if len(es) == 0 {
			t.Fatalf("Expected %#o to be invalid", data)
		}
	}
}

func TestValidateBase64EncodedMap(t *testing.T) {
	validCases := []interface{}{
		map[string]interface{}{
			"key_empty": "", // the plain empty string
		},
		map[string]interface{}{
			"key_encoded_empty": "Cg==", // the encoded empty string
		},
		map[string]interface{}{
			"key_blah": "blah", // "nV"
		},
		map[string]interface{}{
			"key_Test": "VGVzdAo=", // "Test"
		},
	}
	for _, data := range validCases {
		_, es := ValidateBase64EncodedMap(data, "binary_data")
		if len(es) > 0 {
			t.Fatalf("Expected %#o to be valid: %#v", data, es)
		}
	}

	invalidCases := []interface{}{
		nil,
		"bl ah",
		"blahd",
		"C=",
		map[string]interface{}{
			"invalid_nil": nil,
		},
		map[string]string{
			"key_ok":         "VGVzdAo=",
			"invalid_string": "blahd",
		},
		map[string]int{
			"invalid_int": 10,
		},
		map[int]int{
			10: 10,
		},
		map[string]byte{
			"invalid_byte": 10,
		},
	}
	for _, data := range invalidCases {
		_, es := ValidateBase64EncodedMap(data, "binary_data")
		if len(es) == 0 {
			t.Fatalf("Expected %#o to be invalid", data)
		}
	}
}

func TestValidateCronJobFormatting(t *testing.T) {
	testCases := []string{
		"30 04 * * *",
		"TZ=UTC 30 04 * * *",
		"CRON_TZ=UTC 30 04 * * *",
	}
	validator := ValidateCronExpression()
	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			_, err := validator(tc, "spec.0.schedule")
			if len(err) != 0 {
				t.Errorf("Failed to parse cron expression %q: %v", tc, err)
			}
		})
	}
}

func TestValidateNonNegativeInteger(t *testing.T) {
	validCases := []int{
		0,
		1,
		2,
	}
	for _, data := range validCases {
		_, es := ValidateNonNegativeInteger(data, "replicas")
		if len(es) > 0 {
			t.Fatalf("Expected %#o to be valid: %#v", data, es)
		}
	}
	invalidCases := []int{
		-1,
		-2,
		-3,
	}
	for _, data := range invalidCases {
		_, es := ValidateNonNegativeInteger(data, "replicas")
		if len(es) == 0 {
			t.Fatalf("Expected %#o to be invalid", data)
		}
	}
}

func TestValidateTypeStringNullableIntOrPercent(t *testing.T) {
	validCases := []string{
		"",
		"1",
		"100",
		"1%",
		"100%",
	}
	for _, data := range validCases {
		_, es := ValidateTypeStringNullableIntOrPercent(data, "replicas")
		if len(es) > 0 {
			t.Fatalf("Expected %q to be valid: %#v", data, es)
		}
	}
	invalidCases := []string{
		" ",
		"0.1",
		"test",
		"!@@#$",
		"💣",
		"%",
	}
	for _, data := range invalidCases {
		_, es := ValidateTypeStringNullableIntOrPercent(data, "replicas")
		if len(es) == 0 {
			t.Fatalf("Expected %q to be invalid", data)
		}
	}
}
