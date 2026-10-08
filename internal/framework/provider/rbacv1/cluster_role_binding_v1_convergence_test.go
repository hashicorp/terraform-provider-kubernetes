// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func convergenceConfig(name, metadata string) string {
	return strings.Replace(testAccKubernetesClusterRoleBindingV1Config_basic(name),
		fmt.Sprintf("name = %q", name), fmt.Sprintf("name = %q\n%s", name, metadata), 1)
}

func sameBindingUID() resource.TestCheckFunc {
	var prior string
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[clusterRoleBindingResourceName]
		if !ok {
			return fmt.Errorf("missing %s", clusterRoleBindingResourceName)
		}
		uid := rs.Primary.Attributes["metadata.0.uid"]
		if uid == "" {
			return fmt.Errorf("missing ClusterRoleBinding UID")
		}
		if prior != "" && uid != prior {
			return fmt.Errorf("ClusterRoleBinding replaced: UID %q became %q", prior, uid)
		}
		prior = uid
		return nil
	}
}

func TestAccMigrateClusterRoleBindingV1_commonMetadata(t *testing.T) {
	for _, version := range []string{"2.37.1", migratedProviderVersion} {
		for _, empty := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/empty=%t", version, empty), func(t *testing.T) {
				name := "tf-acc-convergence:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
				metadata := ""
				checkPlan := plancheck.ExpectEmptyPlan()
				var mapValue knownvalue.Check = knownvalue.Null()
				if empty {
					metadata = "labels = {}\nannotations = {}"
					checkPlan = plancheck.ExpectResourceAction(clusterRoleBindingResourceName, plancheck.ResourceActionUpdate)
					mapValue = knownvalue.MapExact(map[string]knownvalue.Check{})
				}
				config := convergenceConfig(name, metadata)
				checkUID := sameBindingUID()
				resource.Test(t, resource.TestCase{
					CheckDestroy: testAccKubernetesClusterRoleBindingV1Destroy,
					Steps: []resource.TestStep{
						{
							ExternalProviders: map[string]resource.ExternalProvider{
								"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: version},
							},
							Config: config,
							Check:  checkUID,
						},
						{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Config:                   config,
							Check:                    checkUID,
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply:             []plancheck.PlanCheck{checkPlan},
								PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							},
							ConfigStateChecks: []statecheck.StateCheck{
								statecheck.ExpectKnownValue(clusterRoleBindingResourceName,
									tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("labels"), mapValue),
								statecheck.ExpectKnownValue(clusterRoleBindingResourceName,
									tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("annotations"), mapValue),
								statecheck.ExpectIdentity(clusterRoleBindingResourceName, map[string]knownvalue.Check{
									"api_version": knownvalue.StringExact("rbac.authorization.k8s.io/v1"),
									"kind":        knownvalue.StringExact("ClusterRoleBinding"),
									"name":        knownvalue.StringExact(name),
								}),
							},
						},
					},
				})
			})
		}
	}
}

func TestAccFrameworkClusterRoleBindingV1_commonMetadata(t *testing.T) {
	name := "tf-acc-convergence:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	checkUID := sameBindingUID()
	steps := []resource.TestStep{}
	for _, metadata := range []string{"", "labels = {}\nannotations = {}",
		`labels = { "app.kubernetes.io/name" = "managed" }
annotations = { "example.com/owner" = "terraform" }`, ""} {
		steps = append(steps, resource.TestStep{
			Config: convergenceConfig(name, metadata),
			Check:  checkUID,
		})
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps:                    steps,
	})
}

func TestAccFrameworkClusterRoleBindingV1_commonValidation(t *testing.T) {
	for _, tc := range []struct{ name, metadata, errorPattern string }{
		{"nullLabel", `labels = { x = null }`, "value must be a string"},
		{"nullAnnotation", `annotations = { x = null }`, "value must be a string"},
		{"invalidLabelKey", `labels = { "invalid/key/extra" = "value" }`, "qualified name"},
		{"invalidLabelValue", `labels = { x = "invalid value" }`, "valid label must be an empty string"},
		{"invalidAnnotationKey", `annotations = { "invalid/key/extra" = "value" }`, "qualified name"},
		{"nameConflict", `generate_name = "tf-acc:"`, "cannot be specified when"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := "tf-acc-convergence:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
				Steps: []resource.TestStep{
					{Config: convergenceConfig(name, tc.metadata), ExpectError: regexp.MustCompile(tc.errorPattern)},
					{Config: convergenceConfig(name, "")},
				},
			})
		})
	}
}

func TestAccFrameworkClusterRoleBindingV1_generatedNameMetadataUpdate(t *testing.T) {
	prefix := "tf-acc-convergence:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum) + "-"
	config := testAccKubernetesClusterRoleBindingV1Config_generateName(prefix)
	updated := strings.Replace(config, "metadata {", `metadata {
    labels = { "managed" = "true" }`, 1)
	checkUID := sameBindingUID()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{Config: config, Check: checkUID},
			{
				Config: updated,
				Check:  checkUID,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterRoleBindingResourceName, plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}
