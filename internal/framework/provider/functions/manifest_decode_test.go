package functions

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestManifestDecode(t *testing.T) {
	t.Parallel()

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testARNBuildFunctionConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckOutput("test", "arn:aws:iam::444455556666:role/example"),
				),
			},
		},
	})
}

func testARNBuildFunctionConfig() string {
	return `
output "test" {
  value = provider::kubernetes::manifest_decode(file("testdata/manifest_decode_single.yaml"))
}`
}
