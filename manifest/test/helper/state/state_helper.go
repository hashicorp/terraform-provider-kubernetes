// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance
// +build acceptance

package state

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	tfjson "github.com/hashicorp/terraform-json"
)

// Helper wraps tfjson.State in helper functions for doing assertions in tests
type Helper struct {
	*tfjson.State
}

// NewHelper creates a new state helper
func NewHelper(tfstate *tfjson.State) *Helper {
	return &Helper{tfstate}
}
func (s *Helper) ResourceExists(t *testing.T, resourceAddress string) bool {
	t.Helper()
	_, err := getAttributesValuesFromResource(s, resourceAddress)
	return err == nil
}

// getAttributesValuesFromResource pulls out the AttributeValues field from the resource at the given address
func getAttributesValuesFromResource(state *Helper, address string) (interface{}, error) {
	for _, r := range state.Values.RootModule.Resources {
		if r.Address == address {
			return r.AttributeValues, nil
		}
	}
	return nil, fmt.Errorf("Could not find resource %q in state", address)
}

// getOutputValues gets the given output name value from the state
func getOutputValues(state *Helper, name string) (interface{}, error) {
	for n, v := range state.Values.Outputs {
		if n == name {
			return v.Value, nil
		}
	}

	return nil, fmt.Errorf("Could not find output %q in state", name)
}

var errFieldNotFound = fmt.Errorf("Field not found")

// findAttributeValue will return the value of the attribute at the given address in a tree of arrays and maps
func findAttributeValue(in interface{}, address string) (interface{}, error) {
	keys := strings.Split(address, ".")
	key := keys[0]

	var value interface{}
	if index, err := strconv.Atoi(key); err == nil {
		s, ok := in.([]interface{})
		if !ok || index >= len(s) {
			return nil, errFieldNotFound
		}
		value = s[index]
	} else {
		m, ok := in.(map[string]interface{})
		if !ok {
			return nil, errFieldNotFound
		}
		v, ok := m[key]
		if !ok {
			return nil, errFieldNotFound
		}
		value = v
	}

	if len(keys) == 1 {
		return value, nil
	}

	return findAttributeValue(value, strings.Join(keys[1:], "."))
}

// parseStateAddress will parse an address using the same format as `terraform state show`
// and return the resource address (resource_type.name) and attribute address (attribute.subattribute)
func parseStateAddress(address string) (string, string) {
	parts := strings.Split(address, ".")

	var resourceAddress, attributeAddress string
	switch parts[0] {
	case "data":
		resourceAddress = strings.Join(parts[0:3], ".")
		attributeAddress = strings.Join(parts[3:len(parts)], ".")
	default:
		resourceAddress = strings.Join(parts[0:2], ".")
		attributeAddress = strings.Join(parts[2:len(parts)], ".")
	}

	return resourceAddress, attributeAddress
}

// GetAttributeValue will get the value at the given address from the state
// using the same format as `terraform state show`
func (s *Helper) GetAttributeValue(t *testing.T, address string) interface{} {
	t.Helper()

	resourceAddress, attributeAddress := parseStateAddress(address)
	attrs, err := getAttributesValuesFromResource(s, resourceAddress)
	if err != nil {
		t.Fatal(err)
	}

	value, err := findAttributeValue(attrs, attributeAddress)
	if err != nil {
		t.Fatalf("%q does not exist", address)
	}

	return value
}

// GetOutputValue gets the given output name value from the state
func (s *Helper) GetOutputValue(t *testing.T, name string) interface{} {
	t.Helper()

	value, err := getOutputValues(s, name)
	if err != nil {
		t.Fatal(err)
	}

	return value
}

// AttributeValues is a convenience type for supplying maps of attributes and values
// to AssertAttributeValues
type AttributeValues map[string]interface{}

// AssertAttributeValues will fail the test if the attributes do not have their expected values
func (s *Helper) AssertAttributeValues(t *testing.T, values AttributeValues) {
	t.Helper()

	for address, expectedValue := range values {
		assert.EqualValues(t, expectedValue, s.GetAttributeValue(t, address),
			fmt.Sprintf("Address: %q", address))
	}
}

// AssertAttributeEqual will fail the test if the attribute does not equal expectedValue
func (s *Helper) AssertAttributeEqual(t *testing.T, address string, expectedValue interface{}) {
	t.Helper()

	assert.EqualValues(t, expectedValue, s.GetAttributeValue(t, address),
		fmt.Sprintf("Address: %q", address))
}

// AssertAttributeNotEqual will fail the test if the attribute is equal to expectedValue
func (s *Helper) AssertAttributeNotEqual(t *testing.T, address string, expectedValue interface{}) {
	t.Helper()

	assert.NotEqual(t, expectedValue, s.GetAttributeValue(t, address),
		fmt.Sprintf("Address: %q", address))
}

// AssertAttributeExists will fail the test if the attribute does not exist
func (s *Helper) AssertAttributeExists(t *testing.T, address string) {
	t.Helper()

	s.GetAttributeValue(t, address)
}

// AssertAttributeDoesNotExist will fail the test if the attribute exists
func (s *Helper) AssertAttributeDoesNotExist(t *testing.T, address string) {
	t.Helper()

	resourceAddress, attributeAddress := parseStateAddress(address)
	attrs, err := getAttributesValuesFromResource(s, resourceAddress)
	if err != nil {
		t.Fatal(err)
	}

	_, err = findAttributeValue(attrs, attributeAddress)
	if err == nil {
		t.Fatalf("%q exists", address)
	}
}

// AssertAttributeNotEmpty will fail the test if the attribute is empty
func (s *Helper) AssertAttributeNotEmpty(t *testing.T, address string) {
	t.Helper()

	assert.NotEmpty(t, s.GetAttributeValue(t, address),
		fmt.Sprintf("Address: %q", address))
}

// AssertAttributeEmpty will fail the test if the attribute is not empty
func (s *Helper) AssertAttributeEmpty(t *testing.T, address string) {
	t.Helper()

	assert.Empty(t, s.GetAttributeValue(t, address),
		fmt.Sprintf("Address: %q", address))
}

// AssertAttributeLen will fail the test if the length of the attribute is not exactly length
func (s *Helper) AssertAttributeLen(t *testing.T, address string, length int) {
	t.Helper()

	assert.Len(t, s.GetAttributeValue(t, address), length,
		fmt.Sprintf("Address: %q", address))
}

// AssertAttributeTrue will fail the test if the attribute is not true
func (s *Helper) AssertAttributeTrue(t *testing.T, address string) {
	t.Helper()

	v, ok := s.GetAttributeValue(t, address).(bool)
	if !ok {
		t.Errorf("%q is not a bool", address)
	} else {
		assert.True(t, v, fmt.Sprintf("Address: %q", address))
	}
}

// AssertAttributeFalse will fail the test if the attribute is not false
func (s *Helper) AssertAttributeFalse(t *testing.T, address string) {
	t.Helper()

	v, ok := s.GetAttributeValue(t, address).(bool)
	if !ok {
		t.Errorf("%q is not a bool", address)
	} else {
		assert.False(t, v, fmt.Sprintf("Address: %q", address))
	}
}

// AssertOutputExists will fail the test if the output does not exist
func (s *Helper) AssertOutputExists(t *testing.T, name string) {
	t.Helper()

	s.GetOutputValue(t, name)
}
