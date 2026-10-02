// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
)

func TestCronJobUpdateComputedContainersFollowName(t *testing.T) {
	ctx := context.Background()
	for _, field := range []string{"container", "init_container"} {
		for _, operation := range []string{"remove first", "reorder", "new name"} {
			t.Run(field+"/"+operation, func(t *testing.T) {
				object := cronJobTestObject()
				first := object.Spec.JobTemplate.Spec.Template.Spec.Containers[0]
				first.Name, first.ImagePullPolicy = "first", core.PullAlways
				first.Resources.Requests = core.ResourceList{core.ResourceCPU: apiresource.MustParse("100m")}
				second := *first.DeepCopy()
				second.Name, second.ImagePullPolicy = "second", core.PullNever
				second.Resources.Requests = core.ResourceList{core.ResourceCPU: apiresource.MustParse("200m")}
				if field == "container" {
					object.Spec.JobTemplate.Spec.Template.Spec.Containers = []core.Container{first, second}
				} else {
					object.Spec.JobTemplate.Spec.Template.Spec.InitContainers = []core.Container{first, second}
				}
				_, state := cronJobTestState(t, object)
				desired := []core.Container{second}
				if operation == "reorder" {
					desired = append(desired, first)
				} else if operation == "new name" {
					added := *first.DeepCopy()
					added.Name, added.ImagePullPolicy = "added", ""
					added.Resources = core.ResourceRequirements{}
					desired = []core.Container{added, second}
				}
				changed := object.DeepCopy()
				if field == "container" {
					changed.Spec.JobTemplate.Spec.Template.Spec.Containers = desired
				} else {
					changed.Spec.JobTemplate.Spec.Template.Spec.InitContainers = desired
				}
				_, plan := cronJobTestState(t, changed)
				config := plan
				containersPath := path.Root("spec").AtListIndex(0).AtName("job_template").AtListIndex(0).
					AtName("spec").AtListIndex(0).AtName("template").AtListIndex(0).
					AtName("spec").AtListIndex(0).AtName(field)
				for i := range desired {
					policyPath := containersPath.AtListIndex(i).AtName("image_pull_policy")
					resourcesPath := containersPath.AtListIndex(i).AtName("resources")
					var resources types.List
					if d := plan.GetAttribute(ctx, resourcesPath, &resources); d.HasError() {
						t.Fatal(d)
					}
					for _, d := range []diag.Diagnostics{
						plan.SetAttribute(ctx, policyPath, types.StringUnknown()),
						config.SetAttribute(ctx, policyPath, types.StringNull()),
						plan.SetAttribute(ctx, resourcesPath, types.ListUnknown(resources.ElementType(ctx))),
						config.SetAttribute(ctx, resourcesPath, types.ListNull(resources.ElementType(ctx))),
					} {
						if d.HasError() {
							t.Fatal(d)
						}
					}
				}
				writes := 0
				r := cronJobTestResource(t, func(writer http.ResponseWriter, request *http.Request) {
					switch request.Method {
					case http.MethodGet:
						cronJobTestWriteObject(t, writer, object)
					case http.MethodPut:
						writes++
						var actual batch.CronJob
						if err := json.NewDecoder(request.Body).Decode(&actual); err != nil {
							t.Error(err)
							return
						}
						containers := actual.Spec.JobTemplate.Spec.Template.Spec.Containers
						if field == "init_container" {
							containers = actual.Spec.JobTemplate.Spec.Template.Spec.InitContainers
						}
						if len(containers) != len(desired) {
							t.Errorf("containers=%d, want %d", len(containers), len(desired))
						}
						for i, got := range containers {
							if i >= len(desired) {
								break
							}
							want := desired[i]
							if got.Name != want.Name || got.ImagePullPolicy != want.ImagePullPolicy ||
								!reflect.DeepEqual(got.Resources, want.Resources) {
								t.Errorf("container %s inherited another container's computed values: policy=%s resources=%v; want policy=%s resources=%v",
									got.Name, got.ImagePullPolicy, got.Resources, want.ImagePullPolicy, want.Resources)
							}
						}
						cronJobTestWriteObject(t, writer, &actual)
					default:
						t.Errorf("unexpected request %s", request.Method)
					}
				})
				response := resource.UpdateResponse{State: state}
				r.Update(ctx, resource.UpdateRequest{
					State: state, Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw},
					Config: tfsdk.Config{Schema: config.Schema, Raw: config.Raw},
				}, &response)
				if response.Diagnostics.HasError() {
					t.Fatal(response.Diagnostics)
				}
				if writes != 1 || !response.State.Raw.IsFullyKnown() {
					t.Fatalf("writes=%d, fully known state=%t", writes, response.State.Raw.IsFullyKnown())
				}
			})
		}
	}
}
