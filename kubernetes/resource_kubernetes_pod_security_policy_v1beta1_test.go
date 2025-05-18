// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	policy "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesPodSecurityPolicyV1Beta1_basic(t *testing.T) {
	var conf policy.PodSecurityPolicy
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_pod_security_policy_v1beta1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionGreaterThanOrEqual(t, "1.25.0")
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodSecurityPolicyV1Beta1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodSecurityPolicyV1Beta1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodSecurityPolicyV1Beta1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelFour", "four"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.privileged", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.allow_privilege_escalation", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.host_ipc", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.host_network", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.#", "6"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.0", "configMap"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.1", "emptyDir"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.2", "projected"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.3", "secret"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.4", "downwardAPI"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.5", "persistentVolumeClaim"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.run_as_user.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.run_as_user.0.rule", "MustRunAsNonRoot"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.se_linux.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.se_linux.0.rule", "RunAsAny"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.supplemental_groups.0.rule", "MustRunAs"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.supplemental_groups.0.range.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.supplemental_groups.0.range.0.min", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.supplemental_groups.0.range.0.max", "65535"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.fs_group.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.fs_group.0.rule", "MustRunAs"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.fs_group.0.range.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.fs_group.0.range.0.min", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.fs_group.0.range.0.max", "65535"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.read_only_root_filesystem", "true"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesPodSecurityPolicyV1Beta1Config_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodSecurityPolicyV1Beta1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.privileged", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.allow_privilege_escalation", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.host_ipc", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.host_network", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.#", "6"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.0", "configMap"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.1", "emptyDir"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.2", "projected"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.3", "secret"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.4", "downwardAPI"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.5", "persistentVolumeClaim"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.run_as_user.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.run_as_user.0.rule", "MustRunAsNonRoot"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.se_linux.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.se_linux.0.rule", "RunAsAny"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.supplemental_groups.0.rule", "MustRunAs"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.supplemental_groups.0.range.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.supplemental_groups.0.range.0.min", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.supplemental_groups.0.range.0.max", "65535"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.fs_group.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.fs_group.0.rule", "MustRunAs"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.fs_group.0.range.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.fs_group.0.range.0.min", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.fs_group.0.range.0.max", "65535"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.read_only_root_filesystem", "true"),
				),
			},
			{
				Config: testAccKubernetesPodSecurityPolicyV1Beta1Config_specModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodSecurityPolicyV1Beta1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.privileged", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.allow_privilege_escalation", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.default_allow_privilege_escalation", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.host_ipc", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.host_network", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.allowed_host_paths.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.allowed_host_paths.0.path_prefix", "/"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.allowed_host_paths.0.read_only", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.allowed_unsafe_sysctls.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.allowed_unsafe_sysctls.0", "kernel.msg*"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.forbidden_sysctls.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.forbidden_sysctls.0", "kernel.shm_rmid_forced"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.#", "6"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.0", "configMap"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.1", "emptyDir"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.2", "projected"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.3", "secret"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.4", "downwardAPI"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volumes.5", "persistentVolumeClaim"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.run_as_user.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.run_as_user.0.rule", "MustRunAsNonRoot"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.se_linux.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.se_linux.0.rule", "RunAsAny"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.supplemental_groups.0.rule", "RunAsAny"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.fs_group.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.fs_group.0.rule", "RunAsAny"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.read_only_root_filesystem", "true"),
				),
			},
		},
	})
}

func testAccCheckKubernetesPodSecurityPolicyV1Beta1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_pod_security_policy_v1beta1" {
			continue
		}

		name := rs.Primary.ID

		resp, err := conn.PolicyV1beta1().PodSecurityPolicies().Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == name {
				return fmt.Errorf("Pod Security Policy still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesPodSecurityPolicyV1Beta1Exists(n string, obj *policy.PodSecurityPolicy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()

		name := rs.Primary.ID

		out, err := conn.PolicyV1beta1().PodSecurityPolicies().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesPodSecurityPolicyV1Beta1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_security_policy_v1beta1" "test" {
  metadata {
    name = "%s"

    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
      TestLabelFour  = "four"
    }
  }

  spec {
    volumes = [
      "configMap",
      "emptyDir",
      "projected",
      "secret",
      "downwardAPI",
      "persistentVolumeClaim",
    ]

    run_as_user {
      rule = "MustRunAsNonRoot"
    }

    se_linux {
      rule = "RunAsAny"
    }

    supplemental_groups {
      rule = "MustRunAs"
      range {
        min = 1
        max = 65535
      }
    }

    fs_group {
      rule = "MustRunAs"
      range {
        min = 1
        max = 65535
      }
    }

    host_ports {
      min = 0
      max = 65535
    }

    read_only_root_filesystem = true
  }
}
`, name)
}

func testAccKubernetesPodSecurityPolicyV1Beta1Config_metaModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_security_policy_v1beta1" "test" {
  metadata {
    name = "%s"

    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }
  }

  spec {
    volumes = [
      "configMap",
      "emptyDir",
      "projected",
      "secret",
      "downwardAPI",
      "persistentVolumeClaim",
    ]

    run_as_user {
      rule = "MustRunAsNonRoot"
    }

    se_linux {
      rule = "RunAsAny"
    }

    supplemental_groups {
      rule = "MustRunAs"
      range {
        min = 1
        max = 65535
      }
    }

    fs_group {
      rule = "MustRunAs"
      range {
        min = 1
        max = 65535
      }
    }

    read_only_root_filesystem = true
  }
}
`, name)
}

func testAccKubernetesPodSecurityPolicyV1Beta1Config_specModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_security_policy_v1beta1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    privileged                         = true
    allow_privilege_escalation         = true
    default_allow_privilege_escalation = true
    host_ipc                           = true
    host_network                       = true
    host_pid                           = true

    volumes = [
      "configMap",
      "emptyDir",
      "projected",
      "secret",
      "downwardAPI",
      "persistentVolumeClaim",
    ]

    allowed_host_paths {
      path_prefix = "/"
      read_only   = true
    }

    allowed_unsafe_sysctls = [
      "kernel.msg*"
    ]

    forbidden_sysctls = [
      "kernel.shm_rmid_forced"
    ]

    run_as_user {
      rule = "MustRunAsNonRoot"
    }

    se_linux {
      rule = "RunAsAny"
    }

    supplemental_groups {
      rule = "RunAsAny"
    }

    fs_group {
      rule = "RunAsAny"
    }

    read_only_root_filesystem = true
  }
}
`, name)
}
