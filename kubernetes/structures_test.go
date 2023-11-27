// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"
)

func TestIsInternalKey(t *testing.T) {
	testCases := []struct {
		Key      string
		Expected bool
	}{
		{"", false},
		{"anyKey", false},
		{"any.hostname.io", false},
		{"any.hostname.com/with/path", false},
		{"service.beta.kubernetes.io/aws-load-balancer-backend-protocol", false},
		{"app.kubernetes.io", false},
		{"kubernetes.io", true},
		{"kubectl.kubernetes.io", true},
		{"pv.kubernetes.io/any/path", true},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s", tc.Key), func(t *testing.T) {
			isInternal := isInternalKey(tc.Key)
			if tc.Expected && isInternal != tc.Expected {
				t.Fatalf("Expected %q to be internal", tc.Key)
			}
			if !tc.Expected && isInternal != tc.Expected {
				t.Fatalf("Expected %q not to be internal", tc.Key)
			}
		})
	}
}

func TestPointerOf(t *testing.T) {
	b := false
	bp := pointerOf(b)
	if b != *bp {
		t.Error("Failed to get bool pointer")
	}

	s := "this"
	sp := pointerOf(s)
	if s != *sp {
		t.Error("Failed to get string pointer")
	}

	i := int(1984)
	ip := pointerOf(i)
	if i != *ip {
		t.Error("Failed to get int pointer")
	}

	i64 := int64(1984)
	i64p := pointerOf(i64)
	if i64 != *i64p {
		t.Error("Failed to get int64 pointer")
	}
}
