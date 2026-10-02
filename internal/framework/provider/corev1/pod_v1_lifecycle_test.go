// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	api "k8s.io/api/core/v1"
)

func TestAccKubernetesPodV1_generatedNameLifecycle(t *testing.T) {
	var before, after api.Pod
	resourceName := "kubernetes_pod_v1.test"
	prefix := fmt.Sprintf("tf-acc-test-%s-", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPodV1PreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigGeneratedName(prefix, busyboxImage, "initial"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &before),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
				),
			},
			{
				Config: testAccKubernetesPodV1ConfigGeneratedName(prefix, busyboxImage, "updated"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &after),
					testAccCheckKubernetesPodForceNew(&before, &after, false),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.managed", "updated"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func testAccKubernetesPodV1ConfigGeneratedName(prefix, imageName, label string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    generate_name = %q
    labels = {
      managed = %q
    }
  }

  spec {
    container {
      image = %q
      name  = "containername"
    }
  }
}
`, prefix, label, imageName)
}
