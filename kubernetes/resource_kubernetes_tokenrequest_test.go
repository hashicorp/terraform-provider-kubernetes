// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	api "k8s.io/api/core/v1"
)

func TestAccKubernetesTokenRequest_basic(t *testing.T) {
	var conf api.ServiceAccount
	resourceName := "kubernetes_service_account_v1.test"
	tokenName := "kubernetes_token_request_v1.test"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesTokenRequestConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", "test"),
					resource.TestCheckResourceAttr(tokenName, "metadata.0.name", "test"),
					resource.TestCheckResourceAttr(tokenName, "spec.0.audiences.0", "api"),
					resource.TestCheckResourceAttr(tokenName, "spec.0.audiences.1", "vault"),
					resource.TestCheckResourceAttr(tokenName, "spec.0.audiences.2", "factors"),
					resource.TestCheckResourceAttrSet(tokenName, "token"),
				),
			},
		},
	})
}

func testAccKubernetesTokenRequestConfig_basic() string {
	return fmt.Sprintf(`resource "kubernetes_service_account_v1" "test" {
  metadata {
    name = "test"
  }
}

resource "kubernetes_service_account_v1" "test" {
  metadata {
    name = "test"
  }
}

resource "kubernetes_token_request_v1" "test" {
  metadata {
    name = kubernetes_service_account_v1.test.metadata.0.name
  }
  spec {
    audiences = [
      "api",
      "vault",
      "factors"
    ]
  }
}


`)
}
