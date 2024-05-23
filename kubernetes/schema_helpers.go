// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

func conditionalDefault(condition bool, defaultValue interface{}) interface{} {
	if !condition {
		return nil
	}

	return defaultValue
}
