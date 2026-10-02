// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccDeploymentV1_ReleasedWaitPolicyContract(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-apps-deploy-wait-source")
	config := testAccDeploymentBareContainerConfig(name)
	noWait := strings.Replace(config, "  spec {", "  wait_for_rollout = false\n  spec {", 1)
	address := "kubernetes_deployment_v1.test"
	var snapshot deploymentMigrationSnapshot
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ExternalProviders: map[string]resource.ExternalProvider{
			"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: deploymentSDKv2ProviderVersion},
		},
		CheckDestroy:           testAccCheckKubernetesDeploymentV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "wait_for_rollout", "true"),
					testAccDeploymentMigrationCapture(address, &snapshot),
				),
			},
			{
				Config: noWait,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{deploymentWaitOnlyPlanCheck{before: true, after: false}},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "wait_for_rollout", "false"),
					testAccDeploymentMigrationUnchanged(address, &snapshot),
				),
			},
			{
				ResourceName:    address,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestDeploymentV1WaitOnlyGuard(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mutate  func(*tfjson.ResourceChange)
		wantErr bool
	}{
		{name: "policy-only"},
		{name: "spec-change", mutate: func(change *tfjson.ResourceChange) {
			change.Change.After.(map[string]any)["spec"].([]any)[0].(map[string]any)["replicas"] = "2"
		}, wantErr: true},
		{name: "id-change", mutate: func(change *tfjson.ResourceChange) {
			change.Change.After.(map[string]any)["id"] = "default/replaced"
		}, wantErr: true},
		{name: "replacement", mutate: func(change *tfjson.ResourceChange) {
			change.Change.Actions = tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate}
		}, wantErr: true},
		{name: "unknown-policy", mutate: func(change *tfjson.ResourceChange) {
			change.Change.AfterUnknown = map[string]any{"wait_for_rollout": true}
		}, wantErr: true},
		{name: "no-policy-change", mutate: func(change *tfjson.ResourceChange) {
			change.Change.After.(map[string]any)["wait_for_rollout"] = true
		}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var change tfjson.ResourceChange
			err := json.Unmarshal([]byte(`{
				"address":"kubernetes_deployment_v1.test",
				"change":{
					"actions":["update"],
					"before":{"id":"default/existing","wait_for_rollout":true,"spec":[{"replicas":"1"}]},
					"after":{"id":"default/existing","wait_for_rollout":false,"spec":[{"replicas":"1"}]}
				}
			}`), &change)
			if err != nil {
				t.Fatal(err)
			}
			if tc.mutate != nil {
				tc.mutate(&change)
			}
			var response plancheck.CheckPlanResponse
			deploymentWaitOnlyPlanCheck{before: true, after: false}.CheckPlan(context.Background(), plancheck.CheckPlanRequest{
				Plan: &tfjson.Plan{ResourceChanges: []*tfjson.ResourceChange{&change}},
			}, &response)
			if (response.Error != nil) != tc.wantErr {
				t.Fatalf("unexpected wait-only plan check result: %v", response.Error)
			}
		})
	}
}
