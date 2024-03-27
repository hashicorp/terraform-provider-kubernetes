package functions_test

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestManifestDecode(t *testing.T) {
	t.Parallel()

	outputName := "test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testManifestDecodeConfig("testdata/decode_single.yaml"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue(outputName, knownvalue.ObjectExact(map[string]knownvalue.Check{
						"apiVersion": knownvalue.StringExact("v1"),
						"data": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"configfile": knownvalue.StringExact("---\ntest: document\n"),
						}),
						"kind": knownvalue.StringExact("ConfigMap"),
						"metadata": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"labels": knownvalue.ObjectExact(map[string]knownvalue.Check{
								"test": knownvalue.StringExact("test---label"),
							}),
							"name": knownvalue.StringExact("test-configmap"),
						}),
					})),
				},
			},
		},
	})
}

func testManifestDecodeConfig(filename string) string {
	cwd, _ := os.Getwd()
	return fmt.Sprintf(`
output "test" {
  value = provider::kubernetes::manifest_decode(file(%q))
}`, path.Join(cwd, filename))
}
