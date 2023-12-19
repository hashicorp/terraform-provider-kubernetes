// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package pointer

import (
	"testing"
)

func TestPointerOf(t *testing.T) {
	s := "this"
	sp := PointerOf(s)
	if s != *sp {
		t.Error("Failed to get string pointer")
	}

	i := int(1984)
	ip := PointerOf(i)
	if i != *ip {
		t.Error("Failed to get int pointer")
	}

	i64 := int64(1984)
	i64p := PointerOf(i64)
	if i64 != *i64p {
		t.Error("Failed to get int64 pointer")
	}
}
