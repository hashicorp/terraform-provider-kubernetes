// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package pointer

func PointerOf[A any](a A) *A {
	return &a
}
