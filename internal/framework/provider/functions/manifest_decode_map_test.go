// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package functions_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestManifestDecodeMap(t *testing.T) {
	t.Parallel()

	const first = `
apiVersion: apps/v1
kind: Deployment
metadata: {name: web, namespace: prod}
---
apiVersion: v1
kind: List
items:
- apiVersion: v1
  kind: Namespace
  metadata: {name: prod}
- apiVersion: v1
  kind: ConfigMap
  metadata: {name: settings}
  data: {mode: fast}
`
	const second = `
apiVersion: example.com/v1
kind: Widget
metadata: {name: web, namespace: prod}
`
	call := func(arguments string) string {
		return `output "test" {
  value = provider::kubernetes::manifest_decode_map(` + arguments + `)
}
`
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: call("<<-A\n" + first + "A\n, <<-B\n" + second + "B\n"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.MapPartial(map[string]knownvalue.Check{
						"Deployment.apps/prod/web":    knownvalue.NotNull(),
						"Namespace/prod":              knownvalue.NotNull(),
						"Widget.example.com/prod/web": knownvalue.NotNull(),
						"ConfigMap/settings": knownvalue.ObjectPartial(map[string]knownvalue.Check{
							"data": knownvalue.ObjectExact(map[string]knownvalue.Check{"mode": knownvalue.StringExact("fast")}),
						}),
					})),
					statecheck.ExpectKnownOutputValue("test", knownvalue.MapSizeExact(4)),
				},
			},
			// Read as kubectl reads it: the last line of a block keeps its
			// newline, a large integer stays exact, and the items of a typed
			// list take its kind.
			{
				Config: call("<<-A\napiVersion: v1\nkind: ConfigMap\nmetadata: {name: script, generation: 9007199254740993}\ndata:\n  run: |\n    echo hello\n---\n" +
					"apiVersion: v1\nkind: PodList\nitems:\n- metadata: {name: one}\n---\n" +
					"apiVersion: example.com/v1\nkind: ShoppingList\nmetadata: {name: weekly}\nitems: []\nA\n"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.MapExact(map[string]knownvalue.Check{
						"ConfigMap/script": knownvalue.ObjectPartial(map[string]knownvalue.Check{
							"data":     knownvalue.ObjectExact(map[string]knownvalue.Check{"run": knownvalue.StringExact("echo hello\n")}),
							"metadata": knownvalue.ObjectPartial(map[string]knownvalue.Check{"generation": knownvalue.Int64Exact(9007199254740993)}),
						}),
						"Pod/one":                         knownvalue.ObjectPartial(map[string]knownvalue.Check{"apiVersion": knownvalue.StringExact("v1")}),
						"ShoppingList.example.com/weekly": knownvalue.NotNull(),
					})),
				},
			},
			{
				Config: call("<<-A\napiVersion: v1\nkind: ConfigMap\nmetadata: {name: settings}\n---\n" +
					"apiVersion: v1\nkind: ConfigMap\nmetadata: {name: settings, namespace: default}\nA\n"),
				ExpectError: regexp.MustCompile(`ConfigMap/settings\s+and\s+ConfigMap/default/settings\s+are\s+the\s+same\s+object`),
			},
			{
				Config:      call("<<-A\n" + second + "A\n, <<-B\n" + second + "B\n"),
				ExpectError: regexp.MustCompile(`Widget.example.com/prod/web\s+is\s+defined\s+more\s+than\s+once`),
			},
			{
				Config:      call("<<-A\napiVersion: v1\nkind: ConfigMap\nmetadata: {}\nA\n"),
				ExpectError: regexp.MustCompile(`every\s+object\s+needs\s+apiVersion,\s+kind\s+and\s+metadata.name`),
			},
		},
	})
}
