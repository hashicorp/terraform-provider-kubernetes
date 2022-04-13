package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	policy "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesPodSecurityPolicy_basic(t *testing.T) {
	var conf policy.PodSecurityPolicy
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_pod_security_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodSecurityPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodSecurityPolicyConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodSecurityPolicyExists(resourceName, &conf),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.labels.TestLabelFour", "four"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_pod_security_policy.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_security_policy.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_security_policy.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.privileged", "false"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.allow_privilege_escalation", "false"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.host_ipc", "false"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.host_network", "false"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.#", "6"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.0", "configMap"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.1", "emptyDir"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.2", "projected"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.3", "secret"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.4", "downwardAPI"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.5", "persistentVolumeClaim"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.run_as_user.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.run_as_user.0.rule", "MustRunAsNonRoot"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.se_linux.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.se_linux.0.rule", "RunAsAny"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.supplemental_groups.0.rule", "MustRunAs"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.supplemental_groups.0.range.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.supplemental_groups.0.range.0.min", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.supplemental_groups.0.range.0.max", "65535"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.fs_group.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.fs_group.0.rule", "MustRunAs"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.fs_group.0.range.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.fs_group.0.range.0.min", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.fs_group.0.range.0.max", "65535"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.read_only_root_filesystem", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKubernetesPodSecurityPolicyConfig_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodSecurityPolicyExists(resourceName, &conf),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_pod_security_policy.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_security_policy.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_security_policy.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.privileged", "false"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.allow_privilege_escalation", "false"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.host_ipc", "false"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.host_network", "false"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.#", "6"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.0", "configMap"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.1", "emptyDir"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.2", "projected"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.3", "secret"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.4", "downwardAPI"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.5", "persistentVolumeClaim"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.run_as_user.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.run_as_user.0.rule", "MustRunAsNonRoot"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.se_linux.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.se_linux.0.rule", "RunAsAny"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.supplemental_groups.0.rule", "MustRunAs"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.supplemental_groups.0.range.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.supplemental_groups.0.range.0.min", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.supplemental_groups.0.range.0.max", "65535"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.fs_group.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.fs_group.0.rule", "MustRunAs"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.fs_group.0.range.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.fs_group.0.range.0.min", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.fs_group.0.range.0.max", "65535"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.read_only_root_filesystem", "true"),
				),
			},
			{
				Config: testAccKubernetesPodSecurityPolicyConfig_specModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodSecurityPolicyExists(resourceName, &conf),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_pod_security_policy.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_security_policy.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_security_policy.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.privileged", "true"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.allow_privilege_escalation", "true"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.default_allow_privilege_escalation", "true"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.host_ipc", "true"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.host_network", "true"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.allowed_host_paths.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.allowed_host_paths.0.path_prefix", "/"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.allowed_host_paths.0.read_only", "true"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.allowed_unsafe_sysctls.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.allowed_unsafe_sysctls.0", "kernel.msg*"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.forbidden_sysctls.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.forbidden_sysctls.0", "kernel.shm_rmid_forced"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.#", "6"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.0", "configMap"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.1", "emptyDir"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.2", "projected"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.3", "secret"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.4", "downwardAPI"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.volumes.5", "persistentVolumeClaim"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.run_as_user.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.run_as_user.0.rule", "MustRunAsNonRoot"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.se_linux.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.se_linux.0.rule", "RunAsAny"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.supplemental_groups.0.rule", "RunAsAny"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.fs_group.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.fs_group.0.rule", "RunAsAny"),
					resource.TestCheckResourceAttr("kubernetes_pod_security_policy.test", "spec.0.read_only_root_filesystem", "true"),
				),
			},
		},
	})
}

func testAccCheckKubernetesPodSecurityPolicyDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_pod_security_policy" {
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

func testAccCheckKubernetesPodSecurityPolicyExists(n string, obj *policy.PodSecurityPolicy) resource.TestCheckFunc {
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

func testAccKubernetesPodSecurityPolicyConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_security_policy" "test" {
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

func testAccKubernetesPodSecurityPolicyConfig_metaModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_security_policy" "test" {
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

func testAccKubernetesPodSecurityPolicyConfig_specModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_security_policy" "test" {
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
