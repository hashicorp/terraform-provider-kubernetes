// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Kubernetes stores numbers, so a string argument configured as "01" reads
// back as "1". These keep the configured or prior spelling when it denotes the
// value Kubernetes holds, and otherwise return the value read from Kubernetes.

// KeepIntSpelling keeps prior when both are base-10 integers of equal value.
func KeepIntSpelling(prior, current types.String) types.String {
	return keepSpelling(prior, current, func(a, b string) bool {
		x, err := strconv.ParseInt(a, 10, 64)
		y, err2 := strconv.ParseInt(b, 10, 64)
		return err == nil && err2 == nil && x == y
	})
}

// KeepOctalSpelling keeps prior when both are octal numerals of equal value,
// such as a file mode "00644" that Kubernetes reads back as "0644".
func KeepOctalSpelling(prior, current types.String) types.String {
	return keepSpelling(prior, current, func(a, b string) bool {
		x, err := strconv.ParseInt(a, 8, 64)
		y, err2 := strconv.ParseInt(b, 8, 64)
		return err == nil && err2 == nil && x == y
	})
}

// KeepIntOrStringSpelling keeps prior when both parse to the same IntOrString.
func KeepIntOrStringSpelling(prior, current types.String) types.String {
	return keepSpelling(prior, current, func(a, b string) bool {
		return intstr.Parse(a) == intstr.Parse(b)
	})
}

func keepSpelling(prior, current types.String, same func(a, b string) bool) types.String {
	if prior.IsNull() || prior.IsUnknown() || current.IsNull() || current.IsUnknown() {
		return current
	}
	if same(prior.ValueString(), current.ValueString()) {
		return prior
	}
	return current
}
