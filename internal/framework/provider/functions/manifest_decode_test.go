package functions_test

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
)

func TestManifestDecode(t *testing.T) {
	t.Parallel()

	outputName := "test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testManifestDecodeConfig("testdata/decode_single.yaml"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// FIXME: terraform-plugin-testing doesn't support dynamic yet
					func(s *terraform.State) error {
						ms := s.RootModule()
						rs, ok := ms.Outputs[outputName]
						if !ok {
							return fmt.Errorf("no output value for %q", outputName)
						}
						expectedOutput := map[string]any{
							"apiVersion": "v1",
							"data": map[string]any{
								"configfile": "---\ntest: document\n",
							},
							"kind": "ConfigMap",
							"metadata": map[string]any{
								"labels": map[string]any{
									"test": "test---label",
								},
								"name": "test-configmap",
							},
						}
						assert.Equal(t, expectedOutput, rs.Value)
						return nil
					},
				),
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
