// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"testing"
)

func TestAccKubernetesDataSourceServiceAccountIssuer_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceServiceAccountIssuerConfig(),
				Check: resource.ComposeTestCheckFunc(
					// Ensure issuer and jwks_uri are populated from discovery
					resource.TestCheckResourceAttrSet("data.kubernetes_service_account_issuer.test", "issuer"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service_account_issuer.test", "jwks_uri"),
					// Ensure JWKS payload is fetched and parsed
					resource.TestCheckResourceAttrSet("data.kubernetes_service_account_issuer.test", "jwks_raw"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service_account_issuer.test", "jwks_keys.#"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceServiceAccountIssuerConfig() string {
	return `
data "kubernetes_service_account_issuer" "test" {}
`
}
