package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	api "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesClusterRoleBinding_basic(t *testing.T) {
	var conf api.ClusterRoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesClusterRoleBindingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingExists("kubernetes_cluster_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.kind", "User"),
				),
			},
			{
				Config: testAccKubernetesClusterRoleBindingConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingExists("kubernetes_cluster_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.kind", "User"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.namespace", "kube-system"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.name", "default"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.api_group", ""),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.name", "system:masters"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.kind", "Group"),
				),
			},
			{
				Config: testAccKubernetesClusterRoleBindingConfig_modified_role_ref(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingExists("kubernetes_cluster_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.kind", "User"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.namespace", "kube-system"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.name", "default"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.api_group", ""),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.name", "system:masters"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.kind", "Group"),
				),
			},
			{
				Config: testAccKubernetesClusterRoleBindingConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingExists("kubernetes_cluster_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.kind", "User"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.namespace", "kube-system"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.name", "default"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.api_group", ""),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.name", "system:masters"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.kind", "Group"),
				),
			},
		},
	})
}

func TestAccKubernetesClusterRoleBinding_serviceaccount_subject(t *testing.T) {
	var conf api.ClusterRoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesClusterRoleBindingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingConfig_serviceaccount_subject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingExists("kubernetes_cluster_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.api_group", ""),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.name", "someservice"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.kind", "ServiceAccount"),
				),
			},
		},
	})
}

func TestAccKubernetesClusterRoleBinding_group_subject(t *testing.T) {
	var conf api.ClusterRoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandString(8))
	resourceName := "kubernetes_cluster_role_binding.test"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesClusterRoleBindingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingConfig_group_subject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingExists("kubernetes_cluster_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.name", "somegroup"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.kind", "Group"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKubernetesClusterRoleBinding_UpdatePatchOperationsOrderWithRemovals(t *testing.T) {
	var conf api.ClusterRoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesClusterRoleBindingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingConfigBug_step_0(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingExists("kubernetes_cluster_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.name", "notauser1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.kind", "User"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.name", "notauser2"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.kind", "User"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.name", "notauser3"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.kind", "User"),
				),
			},
			{
				Config: testAccKubernetesClusterRoleBindingConfigBug_step_1(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingExists("kubernetes_cluster_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.name", "notauser2"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.kind", "User"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.name", "notauser4"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.kind", "User"),
				),
			},
			{
				Config: testAccKubernetesClusterRoleBindingConfigBug_step_2(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingExists("kubernetes_cluster_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.#", "4"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.name", "notauser0"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.kind", "User"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.name", "notauser1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.kind", "User"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.name", "notauser2"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.kind", "User"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.3.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.3.name", "notauser3"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.3.kind", "User"),
				),
			},
		},
	})
}

func testAccCheckKubernetesClusterRoleBindingDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_cluster_role_binding" {
			continue
		}
		name := rs.Primary.ID
		resp, err := conn.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("ClusterRoleBinding still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesClusterRoleBindingExists(n string, obj *api.ClusterRoleBinding) resource.TestCheckFunc {
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
		resp, err := conn.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *resp
		return nil
	}
}

func testAccKubernetesClusterRoleBindingConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingConfig_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    api_group = ""
    namespace = "kube-system"
  }

  subject {
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingConfig_modified_role_ref(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    # The kind field only accepts this value, anything else returns an error:
    # roleRef.kind: Unsupported value: "Role": supported values: "ClusterRole"
    kind = "ClusterRole"
    name = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    api_group = ""
    namespace = "kube-system"
  }

  subject {
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingConfig_serviceaccount_subject(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    kind = "ServiceAccount"
    name = "someservice"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingConfig_group_subject(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Group"
    name      = "somegroup"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingConfigBug_step_0(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser1"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser2"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser3"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingConfigBug_step_1(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser2"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser4"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingConfigBug_step_2(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser0"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser1"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser2"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser3"
  }
}
`, name)
}
