// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccKubernetesDataSourceAllNamespaces_basic(t *testing.T) {
	dataSourceName := "data.kubernetes_all_namespaces.test"
	rxPosNum := regexp.MustCompile("^[1-9][0-9]*$")
	rxNsName := regexp.MustCompile(`^[a-zA-Z][-\w]*$`)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAllNamespacesConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					// The cluster always has at least one namespace (default).
					resource.TestMatchResourceAttr(dataSourceName, "namespaces.#", rxPosNum),
					// The first namespace must be set and match the naming regex.
					resource.TestCheckResourceAttrSet(dataSourceName, "namespaces.0"),
					resource.TestMatchResourceAttr(dataSourceName, "namespaces.0", rxNsName),
				),
			},
		},
	})
}

func testAllNamespacesConfig() string {
	return `data "kubernetes_all_namespaces" "test" {}`
}

// TestAccKubernetesDataSourceAllNamespaces_idIsStable covers the one real hazard in this data
// source's design: id is a sha256 over the namespace names in the order the API lists them.
// If that order were unstable, every plan would show a change even with the cluster untouched.
// Two identical steps with an empty plan is what proves it.
//
// Serial for the same reason as the migration test — a namespace appearing between steps
// would change the id legitimately.
func TestAccKubernetesDataSourceAllNamespaces_idIsStable(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAllNamespacesConfig(),
			},
			{
				Config: testAllNamespacesConfig(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
