// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	batchapi "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// All tests in these batch_cli_* files are local Terraform CLI/HTTP QA. They do
// not require TF_ACC, a kubeconfig, or a live cluster, and must not run in parallel
// because Terraform CLI configuration and credentials are process environment.
func batchCLIEnvironment(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Keep workspaces in the repository, but use its root for plugin sockets:
	// the package path plus a socket suffix exceeds macOS's Unix path limit.
	t.Setenv("TMPDIR", filepath.Clean(filepath.Join(cwd, "../../../..")))
	dir := t.TempDir()
	t.Setenv("TF_ACC_TEMP_DIR", dir)
	t.Setenv("HOME", dir)
	t.Setenv("CHECKPOINT_DISABLE", "1")
	t.Setenv("TF_IN_AUTOMATION", "1")
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "KUBE") || strings.HasPrefix(name, "TF_CLI_ARGS") ||
			strings.HasPrefix(name, "TF_VAR_") || name == "TF_REATTACH_PROVIDERS" ||
			name == "TF_PLUGIN_CACHE_DIR" || name == "TF_ACC_TERRAFORM_VERSION" {
			t.Setenv(name, "")
		}
	}
	cli, err := exec.LookPath("terraform")
	if err != nil {
		t.Fatal("local protocol QA requires an installed Terraform CLI:", err)
	}
	t.Setenv("TF_ACC_TERRAFORM_PATH", cli)
	config := filepath.Join(dir, "terraformrc")
	if err := os.WriteFile(config, []byte("disable_checkpoint = true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	// A clean config is essential: dev_overrides silently substitute the local
	// binary even for an ExternalProviders step pinned to a released version.
	t.Setenv("TF_CLI_CONFIG_FILE", config)
	return dir
}

func batchCLIConfig(api *batchCLIAPI, kind string, options batchCLIOptions) string {
	name := `name = "batch-local-qa"`
	if options.generated {
		name = `generate_name = "batch-local-qa-"`
	} else if options.replacement {
		name = `name = "batch-local-qa-replaced"`
	}
	image := "busybox:1.36"
	if options.image != "" {
		image = options.image
	}
	annotations := ""
	if options.annotation != "" {
		annotations = fmt.Sprintf("annotations = { qa = %q }", options.annotation)
	}
	jobFields := ""
	if options.parallelism != 0 {
		jobFields = fmt.Sprintf("parallelism = %d", options.parallelism)
	}
	if options.ttl != "" {
		jobFields += fmt.Sprintf("\n      ttl_seconds_after_finished = %q", options.ttl)
	}
	selector, podFields, resources, init := "", "", "", ""
	if options.rich {
		selector = `
      manual_selector = true
      selector = [{ match_labels = { qa = "batch-local" } }]
`
		podFields = `
          image_pull_secrets = [{ name = "private-registry" }]
          readiness_gate = [{ condition_type = "example.com/ready" }]
          security_context { run_as_non_root = true }
          volume {
            name = "scratch"
            empty_dir {}
          }
`
		resources = `
            resources = [{
              limits = { cpu = "100m", memory = "16Mi" }
              requests = { cpu = "100m", memory = "16Mi" }
            }]
            env {
              name = "QA"
              value = "true"
            }
            volume_mount {
              name = "scratch"
              mount_path = "/scratch"
            }
`
		init = `
          init_container {
            name = "prepare"
            image = "busybox:1.36"
            command = ["sh", "-c", "true"]
            resources = [{
              limits = { cpu = "100m", memory = "16Mi" }
              requests = { cpu = "100m", memory = "16Mi" }
            }]
          }
`
		if options.legacy {
			selector = strings.Replace(selector, `selector = [{ match_labels = { qa = "batch-local" } }]`, `selector { match_labels = { qa = "batch-local" } }`, 1)
			podFields = strings.Replace(podFields, `image_pull_secrets = [{ name = "private-registry" }]`, `image_pull_secrets { name = "private-registry" }`, 1)
			podFields = strings.Replace(podFields, `readiness_gate = [{ condition_type = "example.com/ready" }]`, `readiness_gate { condition_type = "example.com/ready" }`, 1)
			for _, target := range []*string{&resources, &init} {
				*target = strings.ReplaceAll(*target, "resources = [{", "resources {")
				*target = strings.ReplaceAll(*target, "            }]", "            }")
			}
		}
	}
	templateMetadata := "metadata {}"
	if options.rich {
		templateMetadata = `metadata { labels = { qa = "batch-local" } }`
	}
	jobSpec := fmt.Sprintf(`
    spec {
      %s
      %s
      template {
        %s
        spec {
          restart_policy = "Never"
          %s
          %s
          container {
            name = "task"
            image = %q
            command = ["sh", "-c", "true"]
            %s
          }
        }
      }
    }
`, jobFields, selector, templateMetadata, podFields, init, image, resources)
	resourceType := "kubernetes_job_v1"
	if kind == "cronjobs" {
		resourceType = "kubernetes_cron_job_v1"
		schedule := "0 * * * *"
		if options.schedule != "" {
			schedule = options.schedule
		}
		jobSpec = fmt.Sprintf(`
    spec {
      schedule = %q
      job_template {
        metadata {}
        %s
      }
    }
`, schedule, jobSpec)
	}
	return fmt.Sprintf(`
provider "kubernetes" {
  host = %q
}
resource %q "test" {
  metadata {
    %s
    %s
  }
  %s
}
`, api.server.URL, resourceType, name, annotations, jobSpec)
}

type batchCLIOptions struct {
	generated   bool
	replacement bool
	rich        bool
	legacy      bool
	annotation  string
	image       string
	parallelism int
	schedule    string
	ttl         string
}

func batchCLIAddress(kind string) string {
	if kind == "cronjobs" {
		return "kubernetes_cron_job_v1.test"
	}
	return "kubernetes_job_v1.test"
}

func batchCLISettled(checks ...plancheck.PlanCheck) resource.ConfigPlanChecks {
	for index, check := range checks {
		checks[index] = batchCLIPlanDiagnostics{check: check}
	}
	return resource.ConfigPlanChecks{
		PreApply: checks,
		PostApplyPostRefresh: []plancheck.PlanCheck{
			plancheck.ExpectEmptyPlan(),
		},
	}
}

type batchCLIPlanDiagnostics struct {
	check plancheck.PlanCheck
}

func (check batchCLIPlanDiagnostics) CheckPlan(ctx context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	check.check.CheckPlan(ctx, req, resp)
	if resp.Error != nil {
		for _, change := range req.Plan.ResourceChanges {
			if change.Change != nil {
				resp.Error = fmt.Errorf("%w; %s actions=%v replacement_paths=%v",
					resp.Error, change.Address, change.Change.Actions, change.Change.ReplacePaths)
				differences := batchCLIPlanValueChanges("", change.Change.Before, change.Change.After, change.Change.AfterUnknown)
				if len(differences) > 0 {
					resp.Error = fmt.Errorf("%w; known value changes: %s", resp.Error, strings.Join(differences, "; "))
				}
			}
		}
	}
}

func batchCLIPlanValueChanges(path string, before, after, unknown any) []string {
	if unknown == true || reflect.DeepEqual(before, after) {
		return nil
	}
	if old, ok := before.(map[string]any); ok {
		if current, ok := after.(map[string]any); ok {
			unknowns, _ := unknown.(map[string]any)
			keys := make(map[string]bool)
			for key := range old {
				keys[key] = true
			}
			for key := range current {
				keys[key] = true
			}
			names := make([]string, 0, len(keys))
			for key := range keys {
				names = append(names, key)
			}
			sort.Strings(names)
			var differences []string
			for _, key := range names {
				child := key
				if path != "" {
					child = path + "." + key
				}
				differences = append(differences, batchCLIPlanValueChanges(child, old[key], current[key], unknowns[key])...)
			}
			return differences
		}
	}
	if old, ok := before.([]any); ok {
		if current, ok := after.([]any); ok && len(old) == len(current) {
			unknowns, _ := unknown.([]any)
			var differences []string
			for index := range old {
				var unknown any
				if index < len(unknowns) {
					unknown = unknowns[index]
				}
				differences = append(differences, batchCLIPlanValueChanges(
					fmt.Sprintf("%s[%d]", path, index), old[index], current[index], unknown)...)
			}
			return differences
		}
	}
	previous, _ := json.Marshal(before)
	planned, _ := json.Marshal(after)
	return []string{fmt.Sprintf("%s: %s -> %s", path, previous, planned)}
}

func (api *batchCLIAPI) check(kind string, uid *string, creates, updates int, validate func(runtime.Object) error) resource.TestCheckFunc {
	return api.checkState(kind, uid, creates, updates, validate, false)
}

func (api *batchCLIAPI) checkState(kind string, uid *string, creates, updates int, validate func(runtime.Object) error, releasedProvider bool) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		api.mu.Lock()
		defer api.mu.Unlock()
		resourceState, ok := state.RootModule().Resources[batchCLIAddress(kind)]
		if !ok {
			return fmt.Errorf("missing Terraform resource %s", batchCLIAddress(kind))
		}
		_, name, ok := strings.Cut(resourceState.Primary.ID, "/")
		if !ok {
			return fmt.Errorf("invalid namespaced resource ID %q", resourceState.Primary.ID)
		}
		object := api.object(kind, name)
		if object == nil {
			return fmt.Errorf("%s %s is absent from local API", kind, name)
		}
		metadata := batchCLIMetadata(object)
		stateUID := resourceState.Primary.Attributes["metadata.0.uid"]
		// Released Job Create with wait_for_completion=true exits immediately
		// after waiting, before Read populates computed metadata. Capture its API
		// UID now; local migration checks below require the complete state UID.
		unrefreshedReleasedJob := releasedProvider && kind == "jobs" && stateUID == ""
		if metadata.UID == "" || (!unrefreshedReleasedJob && string(metadata.UID) != stateUID) {
			return fmt.Errorf("API UID %q does not match Terraform UID %q", metadata.UID, resourceState.Primary.Attributes["metadata.0.uid"])
		}
		if *uid == "" {
			*uid = string(metadata.UID)
		} else if string(metadata.UID) != *uid {
			return fmt.Errorf("unexpected replacement: UID changed from %s to %s", *uid, metadata.UID)
		}
		if api.calls["POST "+kind] != creates {
			return fmt.Errorf("POST %s: got %d, want %d", kind, api.calls["POST "+kind], creates)
		}
		if got := api.calls["PUT "+kind] + api.calls["PATCH "+kind]; got != updates {
			return fmt.Errorf("updates to %s: got %d, want %d", kind, got, updates)
		}
		if !releasedProvider && api.calls["unversioned PUT "+kind] != 0 {
			return fmt.Errorf("local provider sent an update without the current resourceVersion")
		}
		if validate != nil {
			return validate(object)
		}
		return nil
	}
}

func (api *batchCLIAPI) checkDestroy(_ *terraform.State) error {
	api.mu.Lock()
	defer api.mu.Unlock()
	if len(api.jobs)+len(api.crons) != 0 {
		return fmt.Errorf("destroy left %d Jobs and %d CronJobs", len(api.jobs), len(api.crons))
	}
	for key, count := range api.calls {
		if strings.HasPrefix(key, "unexpected ") {
			return fmt.Errorf("unexpected API endpoint %q reached %d times", key, count)
		}
	}
	for _, kind := range []string{"jobs", "cronjobs"} {
		if api.calls["POST "+kind] != 0 && api.calls["GET "+kind] == 0 {
			return fmt.Errorf("provider never read %s from the local HTTP API", kind)
		}
	}
	return nil
}

func batchCLIUpdated(kind string, options batchCLIOptions) batchCLIOptions {
	if kind == "jobs" {
		options.parallelism = 2
	} else {
		options.schedule = "15 * * * *"
	}
	return options
}

func batchCLIValidateUpdated(object runtime.Object) error {
	switch object := object.(type) {
	case *batchapi.Job:
		if object.Spec.Parallelism == nil || *object.Spec.Parallelism != 2 {
			return fmt.Errorf("Job update did not reach API: parallelism=%v", object.Spec.Parallelism)
		}
	case *batchapi.CronJob:
		if object.Spec.Schedule != "15 * * * *" {
			return fmt.Errorf("CronJob update did not reach API: schedule=%q", object.Spec.Schedule)
		}
	}
	return nil
}

func TestBatchCLI_Lifecycle(t *testing.T) {
	batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		t.Run(kind, func(t *testing.T) {
			api := newBatchCLIAPI(t)
			address := batchCLIAddress(kind)
			initial := batchCLIConfig(api, kind, batchCLIOptions{})
			updatedOptions := batchCLIUpdated(kind, batchCLIOptions{})
			updated := batchCLIConfig(api, kind, updatedOptions)
			replacementOptions := updatedOptions
			if kind == "jobs" {
				replacementOptions.image = "busybox:1.37"
			} else {
				replacementOptions.replacement = true
			}
			replaced := batchCLIConfig(api, kind, replacementOptions)
			var uid, replacementUID string
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             api.checkDestroy,
				Steps: []resource.TestStep{
					{Config: initial, ConfigPlanChecks: batchCLISettled(), Check: api.check(kind, &uid, 1, 0, nil)},
					{Config: initial, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()), Check: api.check(kind, &uid, 1, 0, nil)},
					{
						Config:           updated,
						ConfigPlanChecks: batchCLISettled(plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)),
						Check:            api.check(kind, &uid, 1, 1, batchCLIValidateUpdated),
					},
					{Config: updated, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()), Check: api.check(kind, &uid, 1, 1, nil)},
					{
						ResourceName: address, ImportState: true, ImportStateVerify: true,
					},
					{Config: updated, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()), Check: api.check(kind, &uid, 1, 1, nil)},
					{
						Config:           replaced,
						ConfigPlanChecks: batchCLISettled(plancheck.ExpectResourceAction(address, plancheck.ResourceActionDestroyBeforeCreate)),
						Check: resource.ComposeTestCheckFunc(
							api.check(kind, &replacementUID, 2, 1, nil),
							func(_ *terraform.State) error {
								if replacementUID == uid {
									return fmt.Errorf("replacement retained UID %s", uid)
								}
								return nil
							},
						),
					},
					{Config: replaced, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()), Check: api.check(kind, &replacementUID, 2, 1, nil)},
				},
			})
		})
	}
}

func TestBatchCLI_GeneratedName(t *testing.T) {
	batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		t.Run(kind, func(t *testing.T) {
			api := newBatchCLIAPI(t)
			options := batchCLIOptions{generated: true}
			initial := batchCLIConfig(api, kind, options)
			options.annotation = "unrelated-update"
			updated := batchCLIConfig(api, kind, options)
			var uid string
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             api.checkDestroy,
				Steps: []resource.TestStep{
					{Config: initial, ConfigPlanChecks: batchCLISettled(), Check: api.check(kind, &uid, 1, 0, nil)},
					{
						Config:           updated,
						ConfigPlanChecks: batchCLISettled(plancheck.ExpectResourceAction(batchCLIAddress(kind), plancheck.ResourceActionUpdate)),
						Check: api.check(kind, &uid, 1, 1, func(object runtime.Object) error {
							metadata := batchCLIMetadata(object)
							if !strings.HasPrefix(metadata.Name, "batch-local-qa-") || metadata.Annotations["qa"] != options.annotation {
								return fmt.Errorf("generated name or annotation lost: %v", metadata)
							}
							return nil
						}),
					},
					{Config: updated, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()), Check: api.check(kind, &uid, 1, 1, nil)},
				},
			})
		})
	}
}

func TestBatchCLI_DriftAndDisappearance(t *testing.T) {
	batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		t.Run(kind, func(t *testing.T) {
			api := newBatchCLIAPI(t)
			config := batchCLIConfig(api, kind, batchCLIUpdated(kind, batchCLIOptions{}))
			var uid, recreatedUID string
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             api.checkDestroy,
				Steps: []resource.TestStep{
					{Config: config, Check: api.check(kind, &uid, 1, 0, batchCLIValidateUpdated)},
					{
						PreConfig: func() {
							api.mu.Lock()
							defer api.mu.Unlock()
							api.rv++
							if kind == "jobs" {
								for _, job := range api.jobs {
									*job.Spec.Parallelism = 7
									job.ResourceVersion = strconv.Itoa(api.rv)
								}
							} else {
								for _, cron := range api.crons {
									cron.Spec.Schedule = "30 * * * *"
									cron.ResourceVersion = strconv.Itoa(api.rv)
								}
							}
						},
						Config: config, PlanOnly: true, ExpectNonEmptyPlan: true,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectResourceAction(batchCLIAddress(kind), plancheck.ResourceActionUpdate)},
						},
					},
					{Config: config, ConfigPlanChecks: batchCLISettled(), Check: api.check(kind, &uid, 1, 1, batchCLIValidateUpdated)},
					{
						PreConfig: func() {
							api.mu.Lock()
							defer api.mu.Unlock()
							clear(api.jobs)
							clear(api.crons)
						},
						Config: config, PlanOnly: true, ExpectNonEmptyPlan: true,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectResourceAction(batchCLIAddress(kind), plancheck.ResourceActionCreate)},
						},
					},
					{
						Config: config, ConfigPlanChecks: batchCLISettled(),
						Check: resource.ComposeTestCheckFunc(
							api.check(kind, &recreatedUID, 2, 1, batchCLIValidateUpdated),
							func(_ *terraform.State) error {
								if recreatedUID == uid {
									return fmt.Errorf("recreated object reused deleted UID %s", uid)
								}
								return nil
							},
						),
					},
				},
			})
		})
	}
}

func batchCLIValidateRich(object runtime.Object) error {
	var spec batchapi.JobSpec
	switch object := object.(type) {
	case *batchapi.Job:
		spec = object.Spec
	case *batchapi.CronJob:
		spec = object.Spec.JobTemplate.Spec
	}
	if spec.Selector == nil || spec.Selector.MatchLabels["qa"] != "batch-local" {
		return fmt.Errorf("configured selector did not reach API: %v", spec.Selector)
	}
	pod := spec.Template.Spec
	if len(pod.ImagePullSecrets) != 1 || pod.ImagePullSecrets[0].Name != "private-registry" {
		return fmt.Errorf("image_pull_secrets did not reach API: %v", pod.ImagePullSecrets)
	}
	if len(pod.ReadinessGates) != 1 || pod.ReadinessGates[0].ConditionType != "example.com/ready" {
		return fmt.Errorf("readiness_gate did not reach API: %v", pod.ReadinessGates)
	}
	if len(pod.Containers) != 1 || len(pod.InitContainers) != 1 {
		return fmt.Errorf("containers did not reach API: %v / %v", pod.Containers, pod.InitContainers)
	}
	for _, container := range append(pod.Containers, pod.InitContainers...) {
		if container.Resources.Limits.Cpu().MilliValue() != 100 || container.Resources.Requests.Memory().Value() != 16*1024*1024 {
			return fmt.Errorf("resources for %s did not reach API: %v", container.Name, container.Resources)
		}
	}
	if len(pod.Volumes) != 1 || len(pod.Containers[0].Env) != 1 || len(pod.Containers[0].VolumeMounts) != 1 {
		return fmt.Errorf("regular workload blocks did not reach API")
	}
	return nil
}

func TestBatchCLI_ComputedParentAttributes(t *testing.T) {
	batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		t.Run(kind, func(t *testing.T) {
			api := newBatchCLIAPI(t)
			config := batchCLIConfig(api, kind, batchCLIOptions{rich: true})
			var uid string
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             api.checkDestroy,
				Steps: []resource.TestStep{
					{Config: config, ConfigPlanChecks: batchCLISettled(), Check: api.check(kind, &uid, 1, 0, batchCLIValidateRich)},
					{Config: config, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()), Check: api.check(kind, &uid, 1, 0, batchCLIValidateRich)},
				},
			})
		})
	}
}

func TestBatchCLI_ImportAndPlan(t *testing.T) {
	batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		t.Run(kind, func(t *testing.T) {
			api := newBatchCLIAPI(t)
			config := batchCLIConfig(api, kind, batchCLIOptions{})
			forget := fmt.Sprintf(`
provider "kubernetes" {
  host = %q
}
removed {
  from = %s
  lifecycle {
    destroy = false
  }
}
`, api.server.URL, batchCLIAddress(kind))
			var uid string
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             api.checkDestroy,
				Steps: []resource.TestStep{
					{Config: config, Check: api.check(kind, &uid, 1, 0, nil)},
					{
						ResourceName: batchCLIAddress(kind),
						ImportState:  true, ImportStateVerify: true,
					},
					// plugin-testing 1.13.3 reuses the current state for a
					// persisted CLI import; it does not remove the old address.
					// Forget without destroying, then import into the real
					// working state so the final plan exercises imported state.
					{Config: forget},
					{
						ResourceName: batchCLIAddress(kind), Config: config,
						ImportState: true, ImportStatePersist: true,
						ImportStateId: "default/batch-local-qa",
						ImportStateCheck: func(states []*terraform.InstanceState) error {
							if len(states) != 1 || states[0].ID != "default/batch-local-qa" || states[0].Attributes["metadata.0.uid"] != uid {
								return fmt.Errorf("persistent import did not retain the original API identity: %#v", states)
							}
							return nil
						},
					},
					{
						Config: config, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()),
						Check: api.check(kind, &uid, 1, 0, nil),
					},
				},
			})
		})
	}
}

func TestBatchCLI_OptionalBlockRemoval(t *testing.T) {
	batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		t.Run(kind, func(t *testing.T) {
			api := newBatchCLIAPI(t)
			initial := batchCLIConfig(api, kind, batchCLIOptions{rich: true})
			removed := strings.Replace(initial, `            env {
              name = "QA"
              value = "true"
            }`, "", 1)
			if removed == initial {
				t.Fatal("fixture did not remove the env block")
			}
			// The exact released-provider control below proves CronJob updates
			// its template in place; Job template edits require replacement.
			action, creates, updates := plancheck.ResourceActionUpdate, 1, 1
			var uid, finalUID string
			if kind == "jobs" {
				action, creates, updates = plancheck.ResourceActionDestroyBeforeCreate, 2, 0
			}
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             api.checkDestroy,
				Steps: []resource.TestStep{
					{Config: initial, Check: api.check(kind, &uid, 1, 0, batchCLIValidateRich)},
					{
						Config:           removed,
						ConfigPlanChecks: batchCLISettled(plancheck.ExpectResourceAction(batchCLIAddress(kind), action)),
						Check: api.check(kind, &finalUID, creates, updates, func(object runtime.Object) error {
							var job batchapi.JobSpec
							switch object := object.(type) {
							case *batchapi.Job:
								job = object.Spec
							case *batchapi.CronJob:
								job = object.Spec.JobTemplate.Spec
							}
							if len(job.Template.Spec.Containers[0].Env) != 0 {
								return fmt.Errorf("removed env block remains in API object")
							}
							if (kind == "jobs") == (uid == finalUID) {
								return fmt.Errorf("incorrect UID continuity after env removal: %s -> %s", uid, finalUID)
							}
							return nil
						}),
					},
					{Config: removed, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()), Check: api.check(kind, &finalUID, creates, updates, nil)},
				},
			})
		})
	}
}

func TestBatchCLI_ReleasedOptionalBlockRemoval(t *testing.T) {
	workspaces := batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		t.Run(kind, func(t *testing.T) {
			api := newBatchCLIAPI(t)
			initial := batchCLIConfig(api, kind, batchCLIOptions{rich: true, legacy: true})
			removed := strings.Replace(initial, `            env {
              name = "QA"
              value = "true"
            }`, "", 1)
			if removed == initial {
				t.Fatal("fixture did not remove the env block")
			}
			providers := map[string]resource.ExternalProvider{
				"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
			}
			action := plancheck.ResourceActionDestroyBeforeCreate
			if kind == "cronjobs" {
				action = plancheck.ResourceActionUpdate
			}
			var uid string
			resource.UnitTest(t, resource.TestCase{
				CheckDestroy: api.checkDestroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: providers, Config: initial,
						Check: resource.ComposeTestCheckFunc(
							api.checkState(kind, &uid, 1, 0, batchCLIValidateRich, true),
							batchCLIReleasedVersion(t, workspaces),
						),
					},
					{
						ExternalProviders: providers, Config: removed,
						PlanOnly: true, ExpectNonEmptyPlan: true,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PostApplyPostRefresh: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(batchCLIAddress(kind), action),
							},
						},
					},
				},
			})
		})
	}
}

func TestBatchCLI_Permissions(t *testing.T) {
	batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		t.Run(kind, func(t *testing.T) {
			api := newBatchCLIAPI(t)
			config := batchCLIConfig(api, kind, batchCLIOptions{})
			updated := batchCLIConfig(api, kind, batchCLIUpdated(kind, batchCLIOptions{}))
			deny := func(method string) func() {
				return func() {
					api.mu.Lock()
					defer api.mu.Unlock()
					api.denied = method
					if method != "" {
						api.denied += " " + kind
					}
				}
			}
			updateMethod := "PATCH"
			if kind == "cronjobs" {
				updateMethod = "PUT"
			}
			forbidden := regexp.MustCompile(`(?s)forbidden.*local QA permission denied`)
			var uid string
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             api.checkDestroy,
				Steps: []resource.TestStep{
					{PreConfig: deny("POST"), Config: config, ExpectError: forbidden},
					{PreConfig: deny(""), Config: config, Check: api.check(kind, &uid, 2, 0, nil)},
					{PreConfig: deny("GET"), Config: config, PlanOnly: true, ExpectError: forbidden},
					{PreConfig: deny(""), Config: config, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()), Check: api.check(kind, &uid, 2, 0, nil)},
					{PreConfig: deny(updateMethod), Config: updated, ExpectError: forbidden},
					{PreConfig: deny(""), Config: updated, Check: api.check(kind, &uid, 2, 2, batchCLIValidateUpdated)},
					{PreConfig: deny("DELETE"), Config: updated, Destroy: true, ExpectError: forbidden},
					{PreConfig: deny(""), Config: updated, Destroy: true},
				},
			})
			api.mu.Lock()
			defer api.mu.Unlock()
			if api.calls["DELETE "+kind] < 2 {
				t.Errorf("DELETE permission control did not reach API: %v", api.calls)
			}
		})
	}
}

func TestBatchCLI_UpgradeFrom321(t *testing.T) {
	workspaces := batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		for _, variant := range []string{"minimum", "computed-parent-attributes", "generated-name"} {
			t.Run(kind+"/"+variant, func(t *testing.T) {
				api := newBatchCLIAPI(t)
				options := batchCLIOptions{rich: variant == "computed-parent-attributes", generated: variant == "generated-name"}
				local := batchCLIConfig(api, kind, options)
				options.legacy = true
				legacy := batchCLIConfig(api, kind, options)
				var uid string
				validator := (func(runtime.Object) error)(nil)
				if options.rich {
					validator = batchCLIValidateRich
				}
				validator = batchCLISpecContinuity(validator)
				resource.UnitTest(t, resource.TestCase{
					CheckDestroy: api.checkDestroy,
					Steps: []resource.TestStep{
						{
							ExternalProviders: map[string]resource.ExternalProvider{
								"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
							},
							Config: legacy, Check: resource.ComposeTestCheckFunc(
								api.checkState(kind, &uid, 1, 0, validator, true),
								batchCLIReleasedVersion(t, workspaces),
							),
						},
						{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Config:                   local,
							// The released SDK stores omitted optional scalars as
							// zero/empty. Framework distinguishes null: permit the
							// documented one-time state-only update, but never a
							// replacement, API write, UID or desired-spec change.
							ConfigPlanChecks: batchCLISettled(plancheck.ExpectResourceAction(batchCLIAddress(kind), plancheck.ResourceActionUpdate)),
							Check:            api.check(kind, &uid, 1, 0, validator),
						},
						{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Config:                   local,
							ConfigPlanChecks:         batchCLISettled(plancheck.ExpectEmptyPlan()),
							Check:                    api.check(kind, &uid, 1, 0, validator),
						},
					},
				})
				t.Log("released baseline: hashicorp/kubernetes = 3.2.1; clean CLI config; local migration retained API UID without writes")
			})
		}

	}
}

func batchCLISpecContinuity(validate func(runtime.Object) error) func(runtime.Object) error {
	var baseline []byte
	return func(object runtime.Object) error {
		if validate != nil {
			if err := validate(object); err != nil {
				return err
			}
		}
		var desiredSpec any
		switch object := object.(type) {
		case *batchapi.Job:
			desiredSpec = object.Spec
		case *batchapi.CronJob:
			desiredSpec = object.Spec
		default:
			return fmt.Errorf("unsupported migration object %T", object)
		}
		current, err := json.Marshal(desiredSpec)
		if err != nil {
			return err
		}
		if baseline == nil {
			baseline = current
		} else if !bytes.Equal(baseline, current) {
			return fmt.Errorf("migration changed the API desired spec:\nbefore: %s\nafter: %s", baseline, current)
		}
		return nil
	}
}

func batchCLIReleasedVersion(t *testing.T, workspaces string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		found := false
		err := filepath.WalkDir(workspaces, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Name() != ".terraform.lock.hcl" {
				return nil
			}
			lock, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			pinned := regexp.MustCompile(`(?s)provider "registry.terraform.io/hashicorp/kubernetes" \{\s*version\s*=\s*"3\.2\.1"`)
			if !pinned.Match(lock) {
				return fmt.Errorf("released provider lock does not contain exact hashicorp/kubernetes 3.2.1: %s", path)
			}
			found = true
			t.Log("Terraform dependency lock verified: registry.terraform.io/hashicorp/kubernetes version = 3.2.1")
			return nil
		})
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("released provider dependency lock was not found")
		}
		return nil
	}
}
