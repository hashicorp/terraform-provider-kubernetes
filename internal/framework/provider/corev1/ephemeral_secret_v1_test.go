// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// The echo provider copies the ephemeral values into a resource, where the
// test can read them.
var secretV1EphemeralProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": serviceAccProtoV6ProviderFactories["kubernetes"],
	"echo":       echoprovider.NewProviderServer(),
}

func TestAccKubernetesSecretV1Ephemeral_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	notText := base64.StdEncoding.EncodeToString([]byte{0xff, 0xfe, 0x00})
	read := tfjsonpath.New("data")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: secretV1EphemeralProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesSecretV1EphemeralConfigMissing(name),
				ExpectError: regexp.MustCompile(`Secret not found`),
			},
			{
				Config: testAccKubernetesSecretV1EphemeralConfig(name, notText),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("echo.test", read.AtMapKey("data"), knownvalue.MapExact(map[string]knownvalue.Check{
						"password": knownvalue.StringExact("s3cret"),
					})),
					statecheck.ExpectKnownValue("echo.test", read.AtMapKey("binary_data"), knownvalue.MapExact(map[string]knownvalue.Check{
						"password": knownvalue.StringExact(base64.StdEncoding.EncodeToString([]byte("s3cret"))),
						"blob":     knownvalue.StringExact(notText),
					})),
					statecheck.ExpectKnownValue("echo.test", read.AtMapKey("type"), knownvalue.StringExact("Opaque")),
					statecheck.ExpectKnownValue("echo.test", read.AtMapKey("immutable"), knownvalue.Bool(false)),
				},
			},
		},
	})
}

func testAccKubernetesSecretV1EphemeralConfig(name, notText string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = %[1]q
  }
  data = {
    password = "s3cret"
  }
  binary_data = {
    blob = %[2]q
  }
}

# The id is not known until the Secret exists, so the read waits for it.
ephemeral "kubernetes_secret_v1" "test" {
  metadata {
    name      = split("/", kubernetes_secret_v1.test.id)[1]
    namespace = split("/", kubernetes_secret_v1.test.id)[0]
  }
}

provider "echo" {
  data = ephemeral.kubernetes_secret_v1.test
}

resource "echo" "test" {}
`, name, notText)
}

func testAccKubernetesSecretV1EphemeralConfigMissing(name string) string {
	return fmt.Sprintf(`ephemeral "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s-missing"
  }
}

provider "echo" {
  data = ephemeral.kubernetes_secret_v1.test
}

resource "echo" "test" {}
`, name)
}
