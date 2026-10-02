// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	batchapi "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestBatchCLI_QuantityFormatting(t *testing.T) {
	batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		t.Run(kind, func(t *testing.T) {
			testBatchCLIQuantityFormatting(t, kind, false, "")
		})
	}
}

func TestBatchCLI_ReleasedQuantityFormatting(t *testing.T) {
	workspaces := batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		t.Run(kind, func(t *testing.T) {
			testBatchCLIQuantityFormatting(t, kind, true, workspaces)
		})
	}
}

func testBatchCLIQuantityFormatting(t *testing.T, kind string, released bool, workspaces string) {
	t.Helper()
	api := newBatchCLIAPI(t)
	initial := strings.NewReplacer(`"100m"`, `"1000m"`, `"16Mi"`, `"1024Mi"`).
		Replace(batchCLIConfig(api, kind, batchCLIOptions{rich: true, legacy: released}))
	equivalent := strings.NewReplacer(`"1000m"`, `"1"`, `"1024Mi"`, `"1Gi"`).Replace(initial)
	var uid string
	validate := func(object runtime.Object) error {
		var spec batchapi.JobSpec
		switch object := object.(type) {
		case *batchapi.Job:
			spec = object.Spec
		case *batchapi.CronJob:
			spec = object.Spec.JobTemplate.Spec
		}
		containers := append(spec.Template.Spec.Containers, spec.Template.Spec.InitContainers...)
		for _, container := range containers {
			if container.Resources.Limits.Cpu().MilliValue() != 1000 ||
				container.Resources.Requests.Cpu().MilliValue() != 1000 ||
				container.Resources.Limits.Memory().Value() != 1<<30 ||
				container.Resources.Requests.Memory().Value() != 1<<30 {
				return fmt.Errorf("quantity value changed for %s: %v", container.Name, container.Resources)
			}
		}
		return nil
	}
	validate = batchCLISpecContinuity(validate)
	testCase := resource.TestCase{
		CheckDestroy: api.checkDestroy,
		Steps: []resource.TestStep{
			{Config: initial, ConfigPlanChecks: batchCLISettled(), Check: api.checkState(kind, &uid, 1, 0, validate, released)},
			{
				Config: equivalent,
				// Prove exact released-provider no-op parity through Core,
				// not merely semantic equality of isolated quantity values.
				ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()),
				Check:            api.check(kind, &uid, 1, 0, validate),
			},
			{Config: equivalent, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()), Check: api.check(kind, &uid, 1, 0, validate)},
		},
	}
	if released {
		for index := range testCase.Steps {
			testCase.Steps[index].ExternalProviders = map[string]resource.ExternalProvider{
				"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
			}
		}
		testCase.Steps[0].Check = resource.ComposeTestCheckFunc(testCase.Steps[0].Check, batchCLIReleasedVersion(t, workspaces))
	} else {
		testCase.ProtoV6ProviderFactories = testAccProtoV6ProviderFactories
	}
	resource.UnitTest(t, testCase)
}
