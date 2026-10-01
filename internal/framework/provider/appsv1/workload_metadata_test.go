// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"strings"
	"testing"
)

func withEmptyTemplateNamespace(t *testing.T, config string) string {
	t.Helper()
	template := "\n      metadata {\n"
	if strings.Count(config, template) != 1 {
		t.Fatal("expected exactly one pod-template metadata block")
	}
	return strings.Replace(config, template, template+"        namespace = \"\"\n", 1)
}
