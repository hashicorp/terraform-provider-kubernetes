// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package testdfa

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
)

var muxFactory = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServer(context.Background(), "Test")
	},
}

func TestAccKubernetesDeferredActions_2_step(t *testing.T) {
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_9_0),
		},
		AdditionalCLIOptions: &resource.AdditionalCLIOptions{
			Plan:  resource.PlanOptions{AllowDeferral: true},
			Apply: resource.ApplyOptions{AllowDeferral: true},
		},
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: muxFactory,
				ConfigDirectory: func(tscr config.TestStepConfigRequest) string {
					return "./config-basic"
				},
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("kind_cluster.demo", plancheck.ResourceActionCreate),
						plancheck.ExpectDeferredChange("kubernetes_namespace_v1.demo_ns", plancheck.DeferredReasonProviderConfigUnknown),
						plancheck.ExpectDeferredChange("kubernetes_manifest.crd_workspaces", plancheck.DeferredReasonProviderConfigUnknown),
						plancheck.ExpectDeferredChange("kubernetes_manifest.demo_workspace", plancheck.DeferredReasonProviderConfigUnknown),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("kubernetes_namespace_v1.demo_ns", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("kubernetes_manifest.crd_workspaces", plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("endpoint"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("cluster_ca_certificate"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("client_certificate"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("client_key"), knownvalue.NotNull()),
				},
			},
			{
				ProtoV6ProviderFactories: muxFactory,
				ConfigDirectory: func(tscr config.TestStepConfigRequest) string {
					return "./config-basic"
				},
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("kubernetes_namespace_v1.demo_ns", plancheck.ResourceActionCreate),
						plancheck.ExpectDeferredChange("kubernetes_manifest.demo_workspace", plancheck.DeferredReasonResourceConfigUnknown),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("kubernetes_manifest.demo_workspace", plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("endpoint"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("cluster_ca_certificate"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("client_certificate"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("client_key"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kubernetes_namespace_v1.demo_ns", tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("name"), knownvalue.StringExact("demo-ns")),
					statecheck.ExpectKnownValue("kubernetes_manifest.crd_workspaces", tfjsonpath.New("manifest"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kubernetes_manifest.crd_workspaces", tfjsonpath.New("object"), knownvalue.NotNull()),
				},
			},
			{
				ProtoV6ProviderFactories: muxFactory,
				ConfigDirectory: func(tscr config.TestStepConfigRequest) string {
					return "./config-basic"
				},
				ExpectNonEmptyPlan: false,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNoDeferredChanges(),
						plancheck.ExpectResourceAction("kubernetes_manifest.demo_workspace", plancheck.ResourceActionCreate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("endpoint"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("cluster_ca_certificate"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("client_certificate"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("client_key"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kubernetes_namespace_v1.demo_ns", tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("name"), knownvalue.StringExact("demo-ns")),
					statecheck.ExpectKnownValue("kubernetes_manifest.crd_workspaces", tfjsonpath.New("manifest"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kubernetes_manifest.crd_workspaces", tfjsonpath.New("object"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kubernetes_manifest.demo_workspace", tfjsonpath.New("manifest"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kubernetes_manifest.demo_workspace", tfjsonpath.New("object"), knownvalue.NotNull()),
				},
			},
		},
	})
}
