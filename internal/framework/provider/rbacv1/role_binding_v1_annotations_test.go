// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccRoleBinding_externalAnnotationSubjectUpdate(t *testing.T) {
	for _, migration := range []bool{false, true} {
		t.Run(fmt.Sprintf("migration=%t", migration), func(t *testing.T) {
			name := acctest.RandomWithPrefix("tf-acc-rb")
			resourceName := "kubernetes_role_binding_v1.test"
			var external map[string]resource.ExternalProvider
			factories := testAccMuxProtoV6ProviderFactories
			if migration {
				external = releasedRoleBindingProvider()
				factories = nil
			}
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:               func() { testAccPreCheck(t) },
				CheckDestroy:           testAccRoleBindingCheckDestroy,
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
				Steps: []resource.TestStep{
					{
						ExternalProviders: external, ProtoV6ProviderFactories: factories,
						Config:            roleBindingIgnoreAnnotationConfig + testAccRoleBindingV1Config_users(name, "alice"),
						Check:             testAccRoleBindingExternalAnnotation(name, true),
						ConfigStateChecks: []statecheck.StateCheck{testAccRoleBindingExpectIdentity(resourceName, name)},
					},
					{
						ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
						Config:                   roleBindingIgnoreAnnotationConfig + testAccRoleBindingV1Config_users(name, "alice", "bob"),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						ConfigStateChecks: []statecheck.StateCheck{testAccRoleBindingExpectIdentity(resourceName, name)},
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccRoleBindingExternalAnnotation(name, false),
							testAccRoleBindingCheckLive(name, "", "alice", "bob"),
						),
					},
				},
			})
		})
	}
}

const roleBindingIgnoreAnnotationConfig = `
provider "kubernetes" {
  ignore_annotations = ["external.example.com/note"]
}
`

func TestAccRoleBinding_firstManagedAnnotationPreservesIgnored(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		CheckDestroy:             testAccRoleBindingCheckDestroy,
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: roleBindingIgnoreAnnotationConfig + testAccRoleBindingV1Config_basic(name),
				Check:  testAccRoleBindingExternalAnnotation(name, true),
			},
			{
				Config: roleBindingIgnoreAnnotationConfig + testAccRoleBindingV1Config_updated(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccRoleBindingExternalAnnotation(name, false),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.example.com/note", "updated"),
					resource.TestCheckNoResourceAttr(resourceName, "metadata.0.annotations.external.example.com/note"),
				),
			},
		},
	})
}

func testAccRoleBindingExternalAnnotation(name string, add bool) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		conn, err := testAccRoleClient()
		if err != nil {
			return err
		}
		rb, err := conn.RbacV1().RoleBindings("default").Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if add {
			if rb.Annotations == nil {
				rb.Annotations = map[string]string{}
			}
			rb.Annotations["external.example.com/note"] = "preserve"
			_, err = conn.RbacV1().RoleBindings("default").Update(context.Background(), rb, metav1.UpdateOptions{})
			return err
		}
		if rb.Annotations["external.example.com/note"] != "preserve" {
			return fmt.Errorf("external annotation lost: %v", rb.Annotations)
		}
		return nil
	}
}
