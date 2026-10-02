// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestStatefulSetGeneratedNameFixture(t *testing.T) {
	const prefix = "tf-sts-generated-"
	for name, updated := range map[string]bool{"initial": false, "updated": true} {
		t.Run(name, func(t *testing.T) {
			config := testAccKubernetesStatefulSetV1ConfigGeneratedName(prefix, busyboxImage, updated)
			file, diags := hclsyntax.ParseConfig([]byte(config), "generated.tf", hcl.InitialPos)
			if diags.HasErrors() {
				t.Fatal(diags)
			}
			body := file.Body.(*hclsyntax.Body)
			metadata := body.Blocks[0].Body.Blocks[0].Body
			value, diags := metadata.Attributes["generate_name"].Expr.Value(nil)
			if diags.HasErrors() || value.AsString() != prefix {
				t.Fatalf("generated name prefix changed: %s (%s)", value.GoString(), diags)
			}
			labels, present := metadata.Attributes["labels"]
			if present != updated {
				t.Fatalf("updated labels present = %t, want %t", present, updated)
			}
			if present {
				value, diags := labels.Expr.Value(nil)
				if diags.HasErrors() || value.GetAttr("updated").AsString() != "true" {
					t.Fatalf("updated labels changed: %s (%s)", value.GoString(), diags)
				}
			}
		})
	}
}
