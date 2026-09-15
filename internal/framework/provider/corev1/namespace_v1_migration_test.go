// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// These are the mechanical proof that the migration is behaviour-preserving
// (rule K8S-MIGRATE-015). Everything else in this package is inspection.
//
// Each test creates the namespace with the EXACT released provider, then runs
// the identical configuration against the local build and asserts what the
// practitioner would see. Without this, the first sign of a state-shape mismatch
// is a customer upgrading and seeing a non-empty plan on configuration they
// never touched.

// externalProvider pins the baseline to one exact released version — never an
// operator (rule K8S-MIGRATE-017).
var externalProvider = map[string]resource.ExternalProvider{
	"kubernetes": {VersionConstraint: providerVersion, Source: "hashicorp/kubernetes"},
}

func TestAccKubernetesNamespaceV1_migration_basic(t *testing.T) {
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	var uid string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: externalProvider,
				Config:            testAccKubernetesNamespaceV1Config_basic(nsName),
				Check:             captureNamespaceUID(nsName, &uid),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesNamespaceV1Config_basic(nsName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				// An empty action plan can still hide a refresh-time
				// normalisation, and a migration that quietly recreated the
				// namespace would destroy everything inside it. Assert the
				// remote object is the same one.
				Check: assertNamespaceUIDUnchanged(nsName, &uid),
			},
		},
	})
}

// TestAccKubernetesNamespaceV1_migration_withMetadata covers a namespace that
// actually carries annotations and labels, and anchors `id` into an output so a
// change to the published id attribute would surface as an output diff rather
// than passing silently.
func TestAccKubernetesNamespaceV1_migration_withMetadata(t *testing.T) {
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	var uid string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: externalProvider,
				Config:            testAccKubernetesNamespaceV1Config_migrationWithOutput(nsName),
				Check:             captureNamespaceUID(nsName, &uid),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesNamespaceV1Config_migrationWithOutput(nsName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					assertNamespaceUIDUnchanged(nsName, &uid),
					resource.TestCheckOutput("namespace_id", nsName),
				),
			},
		},
	})
}

// TestAccKubernetesNamespaceV1_migration_generatedName is the case that breaks
// loudest if metadata.name's plan modifiers are ordered wrongly: name is absent
// from the configuration, so it plans unknown, and a RequiresReplace that sees
// that unknown proposes destroying the namespace.
func TestAccKubernetesNamespaceV1_migration_generatedName(t *testing.T) {
	prefix := "tf-acc-test-gen-"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: externalProvider,
				Config:            testAccKubernetesNamespaceV1Config_generatedName(prefix),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesNamespaceV1Config_generatedName(prefix),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// TestAccKubernetesNamespaceV1_migration_deleteTimeout proves the timeouts block
// still decodes: SDKv2 published it as a single nested block containing only
// `delete`, and the Framework block has to render the same object type or state
// written by 3.2.1 fails to read.
func TestAccKubernetesNamespaceV1_migration_deleteTimeout(t *testing.T) {
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: externalProvider,
				Config:            testAccKubernetesNamespaceV1Config_deleteTimeout(nsName),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesNamespaceV1Config_deleteTimeout(nsName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "timeouts.delete", "30m"),
			},
		},
	})
}

// TestAccKubernetesNamespaceV1_migration_explicitEmptyMaps pins the ONE case
// this migration does not keep diff-free, so that it is a tested, documented
// outcome rather than a surprise.
//
// SDKv2 stored `annotations = {}` and an omitted `annotations` identically, so
// the state upgrader cannot tell them apart and nulls both. A practitioner who
// wrote the empty map explicitly therefore sees one in-place update restoring
// it. That update issues no API write — expandMetadata only ever sent a
// non-empty map — and the namespace keeps its UID. The omitted case, which is
// the overwhelmingly common one, stays empty (see _migration_basic above).
func TestAccKubernetesNamespaceV1_migration_explicitEmptyMaps(t *testing.T) {
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	var uid string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: externalProvider,
				Config:            testAccKubernetesNamespaceV1Config_explicitEmptyMaps(nsName),
				Check:             captureNamespaceUID(nsName, &uid),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesNamespaceV1Config_explicitEmptyMaps(nsName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						// Exactly an in-place update — never a replacement.
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: assertNamespaceUIDUnchanged(nsName, &uid),
			},
			{
				// And it converges: the second plan is empty.
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesNamespaceV1Config_explicitEmptyMaps(nsName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func captureNamespaceUID(name string, uid *string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		conn, err := testAccDestroyClientset()
		if err != nil {
			return err
		}
		out, err := conn.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		*uid = string(out.UID)
		return nil
	}
}

func assertNamespaceUIDUnchanged(name string, uid *string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		conn, err := testAccDestroyClientset()
		if err != nil {
			return err
		}
		out, err := conn.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if string(out.UID) != *uid {
			return fmt.Errorf("namespace %s was recreated during the upgrade: uid %s -> %s", name, *uid, out.UID)
		}
		return nil
	}
}

func testAccKubernetesNamespaceV1Config_migrationWithOutput(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne = "one"
    }

    name = "%s"
  }
}

output "namespace_id" {
  value = kubernetes_namespace_v1.test.id
}
`, nsName)
}
