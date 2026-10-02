// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	batchapi "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

func TestBatchCLI_IndexedRetryDefaultsUpgradeFrom321(t *testing.T) {
	workspaces := batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		for _, test := range []struct {
			name    string
			fields  string
			backoff int32
		}{
			{name: "both-omitted"},
			{name: "max-failed-omitted", fields: "backoff_limit_per_index = 2", backoff: 2},
		} {
			t.Run(kind+"/"+test.name, func(t *testing.T) {
				api := newBatchCLIAPI(t)
				config := batchCLIRetryConfig(api, kind, test.fields)
				var uid string
				continuity := batchCLISpecContinuity(batchCLIRetryValues(test.backoff, 0))
				resource.UnitTest(t, resource.TestCase{
					CheckDestroy: api.checkDestroy,
					Steps: []resource.TestStep{
						{
							ExternalProviders: map[string]resource.ExternalProvider{
								"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
							},
							Config: config,
							Check: resource.ComposeAggregateTestCheckFunc(
								api.checkState(kind, &uid, 1, 0, continuity, true),
								batchCLIReleasedVersion(t, workspaces),
							),
						},
						{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Config:                   config,
							ConfigPlanChecks: batchCLISettled(
								plancheck.ExpectResourceAction(batchCLIAddress(kind), plancheck.ResourceActionUpdate),
							),
							Check: api.check(kind, &uid, 1, 0, continuity),
						},
						{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Config:                   config,
							ConfigPlanChecks:         batchCLISettled(plancheck.ExpectEmptyPlan()),
							Check:                    api.check(kind, &uid, 1, 0, continuity),
						},
					},
				})
			})
		}
	}
}

func TestBatchCLI_IndexedBackoffRemoval(t *testing.T) {
	batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		for _, value := range []int32{0, 2} {
			t.Run(fmt.Sprintf("%s/%d", kind, value), func(t *testing.T) {
				api := newBatchCLIAPI(t)
				initial := batchCLIRetryConfig(api, kind, fmt.Sprintf("backoff_limit_per_index = %d", value))
				removed := batchCLIRetryConfig(api, kind, "")
				var beforeUID, afterUID string
				creates := 1
				checkPlan := plancheck.ExpectEmptyPlan()
				if value != 0 {
					creates = 2
					checkPlan = plancheck.ExpectResourceAction(batchCLIAddress(kind), plancheck.ResourceActionDestroyBeforeCreate)
				}
				resource.UnitTest(t, resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					CheckDestroy:             api.checkDestroy,
					Steps: []resource.TestStep{
						{Config: initial, Check: api.check(kind, &beforeUID, 1, 0, batchCLIRetryValues(value, 0))},
						{
							Config: removed, ConfigPlanChecks: batchCLISettled(checkPlan),
							Check: resource.ComposeAggregateTestCheckFunc(
								api.check(kind, &afterUID, creates, 0, batchCLIRetryValues(0, 0)),
								func(_ *terraform.State) error {
									if (beforeUID != afterUID) != (value != 0) {
										return fmt.Errorf("backoff removal %d -> 0: unexpected UID change %s -> %s", value, beforeUID, afterUID)
									}
									return nil
								},
							),
						},
						{
							Config: removed, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()),
							Check: api.check(kind, &afterUID, creates, 0, batchCLIRetryValues(0, 0)),
						},
					},
				})
			})
		}
	}
}

func TestBatchCLI_IndexedRetryDefaultsDrift(t *testing.T) {
	batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		t.Run(kind, func(t *testing.T) {
			api := newBatchCLIAPI(t)
			config := batchCLIRetryConfig(api, kind, "backoff_limit_per_index = 2")
			var uid string
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             api.checkDestroy,
				Steps: []resource.TestStep{
					{Config: config, Check: api.check(kind, &uid, 1, 0, batchCLIRetryValues(2, 0))},
					{
						PreConfig: func() {
							batchCLIMutateJobSpec(t, api, kind, func(spec *batchapi.JobSpec) {
								spec.MaxFailedIndexes = ptr.To(int32(2))
							})
						},
						Config: config,
						ConfigPlanChecks: batchCLISettled(
							plancheck.ExpectResourceAction(batchCLIAddress(kind), plancheck.ResourceActionUpdate),
						),
						Check: api.check(kind, &uid, 1, 1, batchCLIRetryValues(2, 0)),
					},
					{
						Config: config, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()),
						Check: api.check(kind, &uid, 1, 1, batchCLIRetryValues(2, 0)),
					},
				},
			})
		})
	}
}

func TestBatchCLI_IndexedRetryDefaultsImport(t *testing.T) {
	batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		for _, test := range []struct {
			name               string
			fields             string
			backoff, maxFailed int32
		}{
			{name: "zero"},
			{name: "nonzero", fields: "backoff_limit_per_index = 2\nmax_failed_indexes = 1", backoff: 2, maxFailed: 1},
			{name: "absent", backoff: -1, maxFailed: -1},
		} {
			t.Run(kind+"/"+test.name, func(t *testing.T) {
				api := newBatchCLIAPI(t)
				config := batchCLIRetryConfig(api, kind, test.fields)
				forget := fmt.Sprintf(`
provider "kubernetes" { host = %q }
removed {
  from = %s
  lifecycle { destroy = false }
}`, api.server.URL, batchCLIAddress(kind))
				var uid string
				resource.UnitTest(t, resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					CheckDestroy:             api.checkDestroy,
					Steps: []resource.TestStep{
						{Config: config, Check: api.check(kind, &uid, 1, 0, nil)},
						{
							PreConfig: func() {
								if test.name == "absent" {
									batchCLIMutateJobSpec(t, api, kind, func(spec *batchapi.JobSpec) {
										spec.BackoffLimitPerIndex = nil
										spec.MaxFailedIndexes = nil
									})
								}
							},
							Config: forget,
						},
						{
							ResourceName: batchCLIAddress(kind), Config: config,
							ImportState: true, ImportStatePersist: true,
							ImportStateId: "default/batch-local-qa",
							ImportStateCheck: func(states []*terraform.InstanceState) error {
								if len(states) != 1 || states[0].ID != "default/batch-local-qa" || states[0].Attributes["metadata.0.uid"] != uid {
									return fmt.Errorf("retry-limit import did not preserve object identity")
								}
								return nil
							},
						},
						{
							Config: config, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()),
							Check: api.check(kind, &uid, 1, 0, batchCLIRetryValues(test.backoff, test.maxFailed)),
						},
					},
				})
			})
		}
	}
}

func batchCLIMutateJobSpec(t *testing.T, api *batchCLIAPI, kind string, mutate func(*batchapi.JobSpec)) {
	t.Helper()
	api.mu.Lock()
	defer api.mu.Unlock()
	object := api.object(kind, "batch-local-qa")
	spec, err := batchCLIJobSpec(object)
	if err != nil {
		t.Fatal(err)
	}
	mutate(spec)
	api.rv++
	switch object := object.(type) {
	case *batchapi.Job:
		object.ResourceVersion = fmt.Sprint(api.rv)
		api.jobs[object.Name] = object
	case *batchapi.CronJob:
		object.ResourceVersion = fmt.Sprint(api.rv)
		api.crons[object.Name] = object
	}
}

func batchCLIRetryConfig(api *batchCLIAPI, kind, fields string) string {
	config := batchCLIConfig(api, kind, batchCLIOptions{parallelism: 2})
	return strings.Replace(config, "parallelism = 2", fmt.Sprintf(`parallelism = 2
      completion_mode = "Indexed"
      completions = 4
      %s`, fields), 1)
}

func batchCLIRetryValues(backoff, maxFailed int32) func(runtime.Object) error {
	return func(object runtime.Object) error {
		spec, err := batchCLIJobSpec(object)
		if err != nil {
			return err
		}
		if spec.CompletionMode == nil || *spec.CompletionMode != batchapi.IndexedCompletion {
			return fmt.Errorf("Indexed completion mode was lost")
		}
		for _, field := range []struct {
			name string
			got  *int32
			want int32
		}{
			{"backoffLimitPerIndex", spec.BackoffLimitPerIndex, backoff},
			{"maxFailedIndexes", spec.MaxFailedIndexes, maxFailed},
		} {
			if actual := ptr.Deref(field.got, -1); actual != field.want {
				return fmt.Errorf("%s = %d, want %d (-1 means absent)", field.name, actual, field.want)
			}
		}
		return nil
	}
}

func batchCLIJobSpec(object runtime.Object) (*batchapi.JobSpec, error) {
	switch object := object.(type) {
	case *batchapi.Job:
		return &object.Spec, nil
	case *batchapi.CronJob:
		return &object.Spec.JobTemplate.Spec, nil
	default:
		return nil, fmt.Errorf("unexpected batch object %T", object)
	}
}
