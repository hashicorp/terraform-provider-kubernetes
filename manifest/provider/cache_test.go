// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"errors"
	"testing"
)

func TestKeyedCache(t *testing.T) {
	var kc keyedCache[string, string]

	actualVal, actualErr := kc.Get("a", func() (string, error) { return "1", nil })
	if actualErr != nil {
		t.Errorf("unexpected error: %v", actualErr)
	}
	if actualVal != "1" {
		t.Errorf("unexpected value: %s", actualVal)
	}

	actualVal, actualErr = kc.Get("a", func() (string, error) { return "2", nil })
	if actualErr != nil {
		t.Errorf("unexpected error: %v", actualErr)
	}
	if actualVal != "1" {
		t.Errorf("unexpected value: %s", actualVal)
	}

	actualVal, actualErr = kc.Get("b", func() (string, error) { return "2", nil })
	if actualErr != nil {
		t.Errorf("unexpected error: %v", actualErr)
	}
	if actualVal != "2" {
		t.Errorf("unexpected value: %s", actualVal)
	}

	expectedErr := errors.New("something went wrong")
	_, actualErr = kc.Get("c", func() (string, error) { return "", expectedErr })
	if actualErr != expectedErr {
		t.Errorf("unexpected or missing error: %v", actualErr)
	}
}
