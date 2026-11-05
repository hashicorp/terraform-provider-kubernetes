// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package move_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
)

var muxFactory = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServer(context.Background(), "test")
	},
}

func TestAccMoveResourceState_configMapForward(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-move-%s", randomSuffix())
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: muxFactory,
				Config:                   testAccConfigMapConfig_deprecated(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "data.key1", "value1"),
				),
			},
			{
				ProtoV6ProviderFactories: muxFactory,
				Config:                   testAccConfigMapConfig_movedToV1(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("kubernetes_config_map_v1.test", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_config_map_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_config_map_v1.test", "data.key1", "value1"),
				),
			},
		},
	})
}

func TestAccMoveResourceState_configMapReverse(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-move-%s", randomSuffix())
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: muxFactory,
				Config:                   testAccConfigMapConfig_v1(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_config_map_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_config_map_v1.test", "data.key1", "value1"),
				),
			},
			{
				ProtoV6ProviderFactories: muxFactory,
				Config:                   testAccConfigMapConfig_movedToDeprecated(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("kubernetes_config_map.test", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "data.key1", "value1"),
				),
			},
		},
	})
}

func TestAccMoveResourceState_secret(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-move-%s", randomSuffix())
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: muxFactory,
				Config:                   testAccSecretConfig_deprecated(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.name", name),
				),
			},
			{
				ProtoV6ProviderFactories: muxFactory,
				Config:                   testAccSecretConfig_movedToV1(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("kubernetes_secret_v1.test", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_secret_v1.test", "metadata.0.name", name),
				),
			},
		},
	})
}

func TestAccMoveResourceState_daemonset(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-move-%s", randomSuffix())
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: muxFactory,
				Config:                   testAccDaemonsetConfig_deprecated(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.name", name),
				),
			},
			{
				ProtoV6ProviderFactories: muxFactory,
				Config:                   testAccDaemonsetConfig_movedToV1(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("kubernetes_daemon_set_v1.test", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_daemon_set_v1.test", "metadata.0.name", name),
				),
			},
		},
	})
}

// randomSuffix generates a short random string for unique resource names.
func randomSuffix() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 10)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// --- Config helpers ---

func testAccConfigMapConfig_deprecated(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map" "test" {
  metadata {
    namespace = "default"
    name      = "%s"
  }
  data = {
    key1 = "value1"
  }
}
`, name)
}

func testAccConfigMapConfig_v1(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
  metadata {
    namespace = "default"
    name      = "%s"
  }
  data = {
    key1 = "value1"
  }
}
`, name)
}

func testAccConfigMapConfig_movedToDeprecated(name string) string {
	return fmt.Sprintf(`
moved {
  from = kubernetes_config_map_v1.test
  to   = kubernetes_config_map.test
}

resource "kubernetes_config_map" "test" {
  metadata {
    namespace = "default"
    name      = "%s"
  }
  data = {
    key1 = "value1"
  }
}
`, name)
}

func testAccConfigMapConfig_movedToV1(name string) string {
	return fmt.Sprintf(`
moved {
  from = kubernetes_config_map.test
  to   = kubernetes_config_map_v1.test
}

resource "kubernetes_config_map_v1" "test" {
  metadata {
    namespace = "default"
    name      = "%s"
  }
  data = {
    key1 = "value1"
  }
}
`, name)
}

func testAccSecretConfig_deprecated(name string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    namespace = "default"
    name      = "%s"
  }
  data = {
    key1 = "value1"
  }
}
`, name)
}

func testAccSecretConfig_movedToV1(name string) string {
	return fmt.Sprintf(`
moved {
  from = kubernetes_secret.test
  to   = kubernetes_secret_v1.test
}

resource "kubernetes_secret_v1" "test" {
  metadata {
    namespace = "default"
    name      = "%s"
  }
  data = {
    key1 = "value1"
  }
}
`, name)
}

func testAccDaemonsetConfig_deprecated(name string) string {
	return fmt.Sprintf(`resource "kubernetes_daemonset" "test" {
  metadata {
    namespace = "default"
    name      = "%s"
  }
  spec {
    selector {
      match_labels = {
        app = "%s"
      }
    }
    template {
      metadata {
        labels = {
          app = "%s"
        }
      }
      spec {
        container {
          image = "nginx:1.21"
          name  = "nginx"
        }
      }
    }
  }
}
`, name, name, name)
}

func testAccDaemonsetConfig_movedToV1(name string) string {
	return fmt.Sprintf(`
moved {
  from = kubernetes_daemonset.test
  to   = kubernetes_daemon_set_v1.test
}

resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    namespace = "default"
    name      = "%s"
  }
  spec {
    selector {
      match_labels = {
        app = "%s"
      }
    }
    template {
      metadata {
        labels = {
          app = "%s"
        }
      }
      spec {
        container {
          image = "nginx:1.21"
          name  = "nginx"
        }
      }
    }
  }
}
`, name, name, name)
}
