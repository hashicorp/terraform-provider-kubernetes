// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	sdkschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

type cronJobTestClients struct {
	kubernetes.KubeClientsets
	client *k8sclient.Clientset
	err    error
}

func (c cronJobTestClients) MainClientset() (*k8sclient.Clientset, error) { return c.client, c.err }
func (cronJobTestClients) GetIgnoreAnnotations() []string                 { return []string{"^external/"} }
func (cronJobTestClients) GetIgnoreLabels() []string                      { return []string{"^external/"} }

func cronJobTestResource(t *testing.T, handler http.HandlerFunc) *CronJobV1 {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := k8sclient.NewForConfig(&rest.Config{
		Host: server.URL,
		ContentConfig: rest.ContentConfig{
			ContentType: "application/json", AcceptContentTypes: "application/json",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &CronJobV1{SDKv2Meta: func() any { return cronJobTestClients{client: client} }}
}

func cronJobTestObject() *batch.CronJob {
	return &batch.CronJob{
		TypeMeta: metav1.TypeMeta{APIVersion: "batch/v1", Kind: "CronJob"},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test", Namespace: "default", UID: "uid-1", ResourceVersion: "10", Generation: 1,
			Annotations: map[string]string{"managed": "old"},
			Labels:      map[string]string{"managed": "old"},
		},
		Spec: batch.CronJobSpec{
			Schedule: "0 * * * *", ConcurrencyPolicy: batch.AllowConcurrent,
			FailedJobsHistoryLimit: ptr.To(int32(1)), SuccessfulJobsHistoryLimit: ptr.To(int32(3)),
			Suspend: ptr.To(false),
			JobTemplate: batch.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"managed": "old"}},
				Spec: batch.JobSpec{
					BackoffLimit: ptr.To(int32(6)), Parallelism: ptr.To(int32(1)),
					Completions: ptr.To(int32(1)), CompletionMode: ptr.To(batch.NonIndexedCompletion),
					ManualSelector: ptr.To(false),
					Template: core.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"managed": "old"}},
						Spec: core.PodSpec{
							RestartPolicy: core.RestartPolicyNever,
							DNSPolicy:     core.DNSClusterFirst,
							Containers: []core.Container{{
								Name: "task", Image: "busybox:1.36", Command: []string{"true"},
								TerminationMessagePath:   "/dev/termination-log",
								TerminationMessagePolicy: core.TerminationMessageReadFile,
								ImagePullPolicy:          core.PullIfNotPresent,
							}},
						},
					},
				},
			},
		},
	}
}

func cronJobTestState(t *testing.T, object *batch.CronJob) (CronJobV1Model, tfsdk.State) {
	t.Helper()
	ctx := context.Background()
	var response resource.SchemaResponse
	(&CronJobV1{}).Schema(ctx, resource.SchemaRequest{}, &response)
	spec, diagnostics := flattenCronJobSpec(ctx, object.Spec, types.ListNull(cronJobSpecBlock().NestedObject.Type()))
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	meta, diagnostics := common.FlattenNamespacedMetadata(ctx, object.ObjectMeta, nil, nil, nil)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	timeoutType := response.Schema.Blocks["timeouts"].Type().(attr.TypeWithAttributeTypes)
	model := CronJobV1Model{
		ID: types.StringValue(object.Namespace + "/" + object.Name), Metadata: meta, Spec: spec,
		Timeouts: timeouts.Value{Object: types.ObjectNull(timeoutType.AttributeTypes())},
	}
	state := cronJobStateFromModel(t, tfsdk.State{Schema: response.Schema}, model)
	return model, state
}

func cronJobStateFromModel(t *testing.T, prior tfsdk.State, model CronJobV1Model) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: prior.Schema}
	if diagnostics := state.Set(context.Background(), model); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	return state
}

func cronJobTestWriteObject(t *testing.T, writer http.ResponseWriter, object *batch.CronJob) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(object); err != nil {
		t.Error(err)
	}
}

func TestCronJobSchemaLegacyTypesAndVersions(t *testing.T) {
	ctx := context.Background()
	var response resource.SchemaResponse
	r := &CronJobV1{}
	r.Schema(ctx, resource.SchemaRequest{}, &response)
	if response.Schema.Version != 0 {
		t.Fatalf("v1 CronJob schema version changed: %d", response.Schema.Version)
	}
	if _, ok := any(r).(resource.ResourceWithUpgradeState); ok {
		t.Fatal("v1 CronJob has no historical schema versions")
	}
	var identityResponse resource.IdentitySchemaResponse
	r.IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identityResponse)
	if identityResponse.IdentitySchema.Version != 1 {
		t.Fatal("identity version must stay 1")
	}
	legacy, err := sdkschema.NewGRPCProviderServer(kubernetes.Provider()).GetProviderSchema(ctx, &tfprotov5.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	source := legacy.ResourceSchemas["kubernetes_cron_job"].ValueType().(tftypes.Object)
	specList := source.AttributeTypes["spec"].(tftypes.List)
	specObject := specList.ElementType.(tftypes.Object)
	specObject.AttributeTypes["timezone"] = tftypes.String
	specList.ElementType = specObject
	source.AttributeTypes["spec"] = specList
	if !source.Equal(response.Schema.Type().TerraformType(ctx)) {
		t.Fatal("CronJob Framework state type differs from the beta alias plus timezone")
	}
	if cronJobDeleteTimeout != time.Minute {
		t.Fatalf("delete timeout = %s, want 1m", cronJobDeleteTimeout)
	}
}

func TestCronJobScheduleValidation(t *testing.T) {
	for _, test := range []struct {
		value types.String
		valid bool
	}{
		{types.StringNull(), true}, {types.StringUnknown(), true},
		{types.StringValue("@hourly"), true}, {types.StringValue("1 0 * * *"), true},
		{types.StringValue("invalid"), false}, {types.StringValue("* * * * * *"), false},
	} {
		response := validator.StringResponse{}
		cronScheduleValidator{}.ValidateString(context.Background(), validator.StringRequest{
			Path: path.Root("spec").AtListIndex(0).AtName("schedule"), ConfigValue: test.value,
		}, &response)
		if response.Diagnostics.HasError() == test.valid {
			t.Errorf("%s: validation diagnostics %v", test.value, response.Diagnostics)
		}
	}
}

func TestCronJobCreateGeneratedNameAndDefaults(t *testing.T) {
	ctx := context.Background()
	model, state := cronJobTestState(t, cronJobTestObject())
	model.ID = types.StringUnknown()
	model.Metadata[0].Name = types.StringUnknown()
	model.Metadata[0].GenerateName = types.StringValue("test-")
	model.Metadata[0].UID = types.StringUnknown()
	model.Metadata[0].Generation = types.Int64Unknown()
	model.Metadata[0].ResourceVersion = types.StringUnknown()
	plan := cronJobStateFromModel(t, state, model)
	requests := 0
	r := cronJobTestResource(t, func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != http.MethodPost || request.URL.Path != "/apis/batch/v1/namespaces/default/cronjobs" {
			t.Errorf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		var submitted batch.CronJob
		if err := json.NewDecoder(request.Body).Decode(&submitted); err != nil {
			t.Error(err)
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		if submitted.Name != "" || submitted.GenerateName != "test-" ||
			submitted.Spec.FailedJobsHistoryLimit != nil || submitted.Spec.SuccessfulJobsHistoryLimit != nil ||
			submitted.Spec.StartingDeadlineSeconds != nil || submitted.Spec.TimeZone != nil {
			t.Errorf("legacy API omission/default behavior changed: %#v", submitted)
		}
		submitted.Name = "test-generated"
		submitted.UID = "generated-uid"
		submitted.ResourceVersion = "11"
		submitted.Generation = 1
		submitted.Annotations["external/key"] = "server"
		cronJobTestWriteObject(t, writer, &submitted)
	})
	response := resource.CreateResponse{
		State:    tfsdk.State{Schema: state.Schema},
		Identity: &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()},
	}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: state.Schema, Raw: plan.Raw}}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if requests != 1 {
		t.Fatalf("create made %d requests; must use the successful create response", requests)
	}
	var result CronJobV1Model
	if d := response.State.Get(ctx, &result); d.HasError() {
		t.Fatal(d)
	}
	if result.ID.ValueString() != "default/test-generated" || result.Metadata[0].UID.ValueString() != "generated-uid" {
		t.Fatalf("lost generated identity: %#v", result.Metadata)
	}
	if !result.Metadata[0].Annotations.Equal(model.Metadata[0].Annotations) || !result.Spec.Equal(model.Spec) {
		t.Fatal("create overwrote known plan values")
	}
	var identity common.NamespacedResourceIdentity
	if d := response.Identity.Get(ctx, &identity); d.HasError() {
		t.Fatal(d)
	}
	if identity.Name.ValueString() != "test-generated" || identity.APIVersion.ValueString() != "batch/v1" {
		t.Fatalf("unexpected identity: %#v", identity)
	}
}

func TestCronJobCreateRetainsStateAfterSuccessfulRemoteWrite(t *testing.T) {
	ctx := context.Background()
	model, state := cronJobTestState(t, cronJobTestObject())
	plan := cronJobStateFromModel(t, state, model)
	resourcesPath := path.Root("spec").AtListIndex(0).
		AtName("job_template").AtListIndex(0).AtName("spec").AtListIndex(0).
		AtName("template").AtListIndex(0).AtName("spec").AtListIndex(0).
		AtName("container").AtListIndex(0).AtName("resources")
	var resources types.List
	if d := plan.GetAttribute(ctx, resourcesPath, &resources); d.HasError() {
		t.Fatal(d)
	}
	if d := plan.SetAttribute(ctx, resourcesPath, types.ListUnknown(resources.ElementType(ctx))); d.HasError() {
		t.Fatal(d)
	}
	r := cronJobTestResource(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("unexpected request %s", request.Method)
		}
		out := cronJobTestObject()
		out.Spec.JobTemplate.Spec.Template.Spec.Containers = append(out.Spec.JobTemplate.Spec.Template.Spec.Containers,
			core.Container{Name: "injected", Image: "busybox:1.36"})
		cronJobTestWriteObject(t, writer, out)
	})
	response := resource.CreateResponse{State: tfsdk.State{Schema: state.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: state.Schema, Raw: plan.Raw}}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("unexpected API collection shape must report an error")
	}
	var id string
	if d := response.State.GetAttribute(ctx, path.Root("id"), &id); d.HasError() || id != "default/test" {
		t.Fatalf("successful remote create lost ID: %q, %v", id, d)
	}
}

func TestCronJobUpdatePreservesServerMetadata(t *testing.T) {
	ctx := context.Background()
	old := cronJobTestObject()
	old.GenerateName = "test-"
	model, state := cronJobTestState(t, old)
	desired := old.DeepCopy()
	desired.Spec.Schedule = "@daily"
	desired.Spec.FailedJobsHistoryLimit = ptr.To(int32(0))
	desired.Spec.SuccessfulJobsHistoryLimit = ptr.To(int32(0))
	desired.Spec.Suspend = ptr.To(true)
	desired.Annotations = nil
	desired.Labels = map[string]string{"new": "value"}
	desired.Spec.JobTemplate.Labels = nil
	desired.Spec.JobTemplate.Spec.Template.Annotations = nil
	model.Metadata[0].Annotations = types.MapNull(types.StringType)
	model.Metadata[0].Labels = types.MapValueMust(types.StringType, map[string]attr.Value{"new": types.StringValue("value")})
	newSpec, d := flattenCronJobSpec(ctx, desired.Spec, model.Spec)
	if d.HasError() {
		t.Fatal(d)
	}
	model.Spec = newSpec
	plan := cronJobStateFromModel(t, state, model)
	current := old.DeepCopy()
	current.ResourceVersion = "20"
	current.Annotations["external/key"] = "annotation"
	current.Labels["external/key"] = "label"
	current.Spec.JobTemplate.Labels["external/key"] = "job"
	current.Spec.JobTemplate.Spec.Template.Annotations["external/key"] = "pod"
	current.Finalizers = []string{"example.com/controller"}
	requests := []string{}
	r := cronJobTestResource(t, func(writer http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.Method)
		if request.URL.Path != "/apis/batch/v1/namespaces/default/cronjobs/test" {
			t.Errorf("unexpected URL %s", request.URL.Path)
		}
		if request.Method == http.MethodGet {
			cronJobTestWriteObject(t, writer, current)
			return
		}
		if request.Method != http.MethodPut {
			t.Errorf("unexpected method %s", request.Method)
		}
		var submitted batch.CronJob
		if err := json.NewDecoder(request.Body).Decode(&submitted); err != nil {
			t.Error(err)
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		if submitted.ResourceVersion != "20" || submitted.GenerateName != "test-" || submitted.Name != "test" ||
			!reflect.DeepEqual(submitted.Finalizers, current.Finalizers) {
			t.Errorf("server metadata overwritten: %#v", submitted.ObjectMeta)
		}
		if submitted.Annotations["external/key"] != "annotation" || submitted.Labels["external/key"] != "label" ||
			submitted.Spec.JobTemplate.Labels["external/key"] != "job" ||
			submitted.Spec.JobTemplate.Spec.Template.Annotations["external/key"] != "pod" {
			t.Error("unowned metadata was erased")
		}
		if _, exists := submitted.Annotations["managed"]; exists {
			t.Error("removed managed annotation still exists")
		}
		if submitted.Spec.Schedule != "@daily" || *submitted.Spec.FailedJobsHistoryLimit != 0 ||
			*submitted.Spec.SuccessfulJobsHistoryLimit != 0 || !*submitted.Spec.Suspend {
			t.Errorf("planned spec not submitted: %#v", submitted.Spec)
		}
		submitted.ResourceVersion = "21"
		cronJobTestWriteObject(t, writer, &submitted)
	})
	response := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: tfsdk.Plan{Schema: state.Schema, Raw: plan.Raw}}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if !reflect.DeepEqual(requests, []string{http.MethodGet, http.MethodPut}) {
		t.Fatalf("unexpected requests: %v", requests)
	}
	var result CronJobV1Model
	if d := response.State.Get(ctx, &result); d.HasError() {
		t.Fatal(d)
	}
	if !result.Spec.Equal(model.Spec) || !result.Metadata[0].Labels.Equal(model.Metadata[0].Labels) ||
		result.Metadata[0].ResourceVersion.ValueString() != "21" {
		t.Fatal("update failed to retain plan and new resourceVersion")
	}
}

func TestCronJobUpdateRepresentationOnlyDoesNotWrite(t *testing.T) {
	for _, change := range []string{"legacy empty strings", "timeouts only"} {
		t.Run(change, func(t *testing.T) {
			ctx := context.Background()
			model, state := cronJobTestState(t, cronJobTestObject())
			if change == "legacy empty strings" {
				for _, field := range []path.Path{
					path.Root("metadata").AtListIndex(0).AtName("generate_name"),
					path.Root("spec").AtListIndex(0).AtName("job_template").AtListIndex(0).
						AtName("metadata").AtListIndex(0).AtName("generate_name"),
					path.Root("spec").AtListIndex(0).AtName("job_template").AtListIndex(0).
						AtName("spec").AtListIndex(0).AtName("template").AtListIndex(0).
						AtName("metadata").AtListIndex(0).AtName("generate_name"),
				} {
					if d := state.SetAttribute(ctx, field, types.StringValue("")); d.HasError() {
						t.Fatal(d)
					}
				}
			} else {
				model.Timeouts = timeouts.Value{Object: types.ObjectValueMust(model.Timeouts.AttributeTypes(ctx),
					map[string]attr.Value{"delete": types.StringValue("2m")})}
			}
			plan := cronJobStateFromModel(t, state, model)
			requests := []string{}
			r := cronJobTestResource(t, func(writer http.ResponseWriter, request *http.Request) {
				requests = append(requests, request.Method)
				if request.Method != http.MethodGet {
					t.Errorf("representation-only update must not write: %s", request.Method)
				}
				cronJobTestWriteObject(t, writer, cronJobTestObject())
			})
			response := resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{
				State: state, Plan: tfsdk.Plan{Schema: state.Schema, Raw: plan.Raw},
			}, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			if !reflect.DeepEqual(requests, []string{http.MethodGet}) {
				t.Fatalf("representation-only update requests: %v", requests)
			}
			if !response.State.Raw.Equal(plan.Raw) {
				t.Fatal("state-only update did not preserve the normalized plan")
			}
		})
	}
}

func TestCronJobComparisonPayloadEquivalence(t *testing.T) {
	ctx := context.Background()
	_, previous := cronJobTestState(t, cronJobTestObject())
	template := path.Root("spec").AtListIndex(0).AtName("job_template").AtListIndex(0).
		AtName("spec").AtListIndex(0).AtName("template")
	generateName := template.AtListIndex(0).AtName("metadata").AtListIndex(0).AtName("generate_name")
	container := template.AtListIndex(0).AtName("spec").AtListIndex(0).AtName("container").AtListIndex(0)
	image := container.AtName("image")
	for _, test := range []struct {
		name       string
		change     func(*tfsdk.State)
		equivalent bool
	}{
		{name: "legacy empty to null", equivalent: true},
		{name: "image edit", change: func(plan *tfsdk.State) {
			if d := plan.SetAttribute(ctx, image, types.StringValue("busybox:next")); d.HasError() {
				t.Fatal(d)
			}
		}},
		{name: "configured unknown", change: func(plan *tfsdk.State) {
			if d := plan.SetAttribute(ctx, image, types.StringUnknown()); d.HasError() {
				t.Fatal(d)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			prior, plan := previous, previous
			if d := prior.SetAttribute(ctx, generateName, types.StringValue("")); d.HasError() {
				t.Fatal(d)
			}
			if test.change != nil {
				test.change(&plan)
			}
			before := plan.Raw.Copy()
			equivalent := jobSpecPayloadEquivalent(ctx,
				tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}, prior,
				tfsdk.Config{Schema: plan.Schema, Raw: plan.Raw})
			if equivalent != test.equivalent {
				t.Fatalf("payload equivalent=%t, want %t", equivalent, test.equivalent)
			}
			if !plan.Raw.Equal(before) {
				t.Fatal("payload comparator rewrote the real plan")
			}
		})
	}
}

func TestCronJobUpdateComputedUnknownsDoNotWrite(t *testing.T) {
	ctx := context.Background()
	_, state := cronJobTestState(t, cronJobTestObject())
	plan, config := state, state
	templatePath := path.Root("spec").AtListIndex(0).AtName("job_template").AtListIndex(0).
		AtName("spec").AtListIndex(0).AtName("template")
	containerPath := templatePath.AtListIndex(0).AtName("spec").AtListIndex(0).AtName("container").AtListIndex(0)
	for _, field := range []path.Path{
		containerPath.AtName("image_pull_policy"),
		templatePath.AtListIndex(0).AtName("metadata").AtListIndex(0).AtName("name"),
	} {
		if d := plan.SetAttribute(ctx, field, types.StringUnknown()); d.HasError() {
			t.Fatal(d)
		}
		if d := config.SetAttribute(ctx, field, types.StringNull()); d.HasError() {
			t.Fatal(d)
		}
	}
	generateName := templatePath.AtListIndex(0).AtName("metadata").AtListIndex(0).AtName("generate_name")
	if d := state.SetAttribute(ctx, generateName, types.StringValue("")); d.HasError() {
		t.Fatal(d)
	}
	requests := []string{}
	r := cronJobTestResource(t, func(writer http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.Method)
		if request.Method != http.MethodGet {
			t.Errorf("unconfigured computed unknowns must not cause a write: %s", request.Method)
		}
		cronJobTestWriteObject(t, writer, cronJobTestObject())
	})
	response := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{
		State: state, Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw},
		Config: tfsdk.Config{Schema: config.Schema, Raw: config.Raw},
	}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if !reflect.DeepEqual(requests, []string{http.MethodGet}) || !response.State.Raw.IsFullyKnown() {
		t.Fatalf("normalization with computed unknowns requests=%v, fully known=%t", requests, response.State.Raw.IsFullyKnown())
	}
	var imagePullPolicy types.String
	if d := response.State.GetAttribute(ctx, containerPath.AtName("image_pull_policy"), &imagePullPolicy); d.HasError() ||
		imagePullPolicy.ValueString() != "IfNotPresent" {
		t.Fatalf("computed API value not persisted: %s, %v", imagePullPolicy, d)
	}
}

func TestCronJobReadFiltersAndAbsence(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNotFound, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			ctx := context.Background()
			old := cronJobTestObject()
			model, state := cronJobTestState(t, old)
			actual := old.DeepCopy()
			actual.Spec.Schedule = "@weekly"
			actual.Annotations["external/key"] = "ignore"
			actual.Labels["external/key"] = "ignore"
			requests := 0
			r := cronJobTestResource(t, func(writer http.ResponseWriter, request *http.Request) {
				requests++
				if request.Method != http.MethodGet {
					t.Errorf("unexpected write on read: %s", request.Method)
				}
				if status != http.StatusOK {
					writer.WriteHeader(status)
					return
				}
				cronJobTestWriteObject(t, writer, actual)
			})
			response := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &response)
			if requests != 1 {
				t.Fatalf("Read made %d requests", requests)
			}
			switch status {
			case http.StatusNotFound:
				if response.Diagnostics.HasError() || !response.State.Raw.IsNull() {
					t.Fatal("NotFound must remove state")
				}
			case http.StatusForbidden:
				if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) {
					t.Fatal("read error must retain state")
				}
			default:
				if response.Diagnostics.HasError() {
					t.Fatal(response.Diagnostics)
				}
				var result CronJobV1Model
				if d := response.State.Get(ctx, &result); d.HasError() {
					t.Fatal(d)
				}
				if !result.Metadata[0].Annotations.Equal(model.Metadata[0].Annotations) ||
					result.Spec.Equal(model.Spec) {
					t.Fatal("read failed filtering or failed to detect schedule drift")
				}
			}
		})
	}
}

func TestCronJobDeleteAbsenceErrorsAndTimeout(t *testing.T) {
	for _, test := range []struct {
		name         string
		deleteStatus int
		getStatus    int
		timeout      string
		wantError    bool
	}{
		{"already absent", http.StatusNotFound, 0, "", false},
		{"deleted", http.StatusOK, http.StatusNotFound, "", false},
		{"forbidden", http.StatusForbidden, 0, "", true},
		{"read failure", http.StatusOK, http.StatusForbidden, "", true},
		{"timeout", http.StatusOK, http.StatusOK, "10ms", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			model, state := cronJobTestState(t, cronJobTestObject())
			if test.timeout != "" {
				model.Timeouts = timeouts.Value{Object: types.ObjectValueMust(model.Timeouts.AttributeTypes(ctx),
					map[string]attr.Value{"delete": types.StringValue(test.timeout)})}
				state = cronJobStateFromModel(t, state, model)
			}
			requests := []string{}
			r := cronJobTestResource(t, func(writer http.ResponseWriter, request *http.Request) {
				requests = append(requests, request.Method)
				status := test.getStatus
				if request.Method == http.MethodDelete {
					status = test.deleteStatus
				}
				if status == http.StatusOK {
					cronJobTestWriteObject(t, writer, cronJobTestObject())
					return
				}
				writer.WriteHeader(status)
			})
			response := resource.DeleteResponse{State: state}
			r.Delete(ctx, resource.DeleteRequest{State: state}, &response)
			if response.Diagnostics.HasError() != test.wantError {
				t.Fatalf("diagnostics: %v", response.Diagnostics)
			}
			if len(requests) == 0 || requests[0] != http.MethodDelete {
				t.Fatalf("unexpected requests %v", requests)
			}
			if test.getStatus == 0 && len(requests) != 1 {
				t.Fatalf("delete error/absence unexpectedly polled: %v", requests)
			}
			if test.wantError && !response.State.Raw.Equal(state.Raw) {
				t.Fatal("failed delete lost state")
			}
		})
	}
}

func TestCronJobConfigureErrors(t *testing.T) {
	ctx := context.Background()
	_, state := cronJobTestState(t, cronJobTestObject())
	for _, metadata := range []func() any{
		nil, func() any { return "wrong type" },
		func() any { return cronJobTestClients{err: errors.New("client unavailable")} },
	} {
		r := &CronJobV1{SDKv2Meta: metadata}
		response := resource.ReadResponse{State: state}
		r.Read(ctx, resource.ReadRequest{State: state}, &response)
		if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) {
			t.Fatalf("provider/client failure must diagnose without losing state: %v", response.Diagnostics)
		}
	}
	var response resource.ConfigureResponse
	(&CronJobV1{}).Configure(ctx, resource.ConfigureRequest{ProviderData: "wrong type"}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("wrong provider callback must produce diagnostic")
	}
}

func TestCronJobImport(t *testing.T) {
	ctx := context.Background()
	_, state := cronJobTestState(t, cronJobTestObject())
	emptyState := tfsdk.State{Schema: state.Schema, Raw: tftypes.NewValue(state.Schema.Type().TerraformType(ctx), nil)}
	identitySchema := common.NamespacedIdentitySchema()
	nullIdentity := &tfsdk.ResourceIdentity{
		Schema: identitySchema,
		Raw:    tftypes.NewValue(identitySchema.Type().TerraformType(ctx), nil),
	}
	for _, id := range []string{"default/test", "", "test", "/test", "default/", "too/many/parts"} {
		response := resource.ImportStateResponse{State: emptyState}
		(&CronJobV1{}).ImportState(ctx, resource.ImportStateRequest{ID: id, Identity: nullIdentity}, &response)
		valid := id == "default/test"
		if response.Diagnostics.HasError() == valid {
			t.Errorf("%q import diagnostics: %v", id, response.Diagnostics)
		}
	}
	for _, apiVersion := range []string{"batch/v1", "batch/v1beta1"} {
		identity := &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()}
		value := cronJobIdentity("", "test")
		value.APIVersion = types.StringValue(apiVersion)
		value.Namespace = types.StringNull()
		if d := identity.Set(ctx, value); d.HasError() {
			t.Fatal(d)
		}
		response := resource.ImportStateResponse{State: emptyState}
		(&CronJobV1{}).ImportState(ctx, resource.ImportStateRequest{Identity: identity}, &response)
		if apiVersion != "batch/v1" {
			if !response.Diagnostics.HasError() {
				t.Fatal("beta identity must not be accepted for a v1 import")
			}
			continue
		}
		if response.Diagnostics.HasError() {
			t.Fatal(response.Diagnostics)
		}
		var id string
		if d := response.State.GetAttribute(ctx, path.Root("id"), &id); d.HasError() || id != "default/test" {
			t.Fatalf("identity import ID=%q diagnostics=%v", id, d)
		}
	}
}

func TestCronJobUpdateErrorsRetainState(t *testing.T) {
	for _, failure := range []string{http.MethodGet, http.MethodPut} {
		t.Run(failure, func(t *testing.T) {
			model, state := cronJobTestState(t, cronJobTestObject())
			changed := cronJobTestObject()
			changed.Spec.Schedule = "@daily"
			plannedSpec, d := flattenCronJobSpec(context.Background(), changed.Spec, model.Spec)
			if d.HasError() {
				t.Fatal(d)
			}
			model.Spec = plannedSpec
			plan := cronJobStateFromModel(t, state, model)
			r := cronJobTestResource(t, func(writer http.ResponseWriter, request *http.Request) {
				if request.Method == failure {
					http.Error(writer, "denied", http.StatusForbidden)
					return
				}
				cronJobTestWriteObject(t, writer, cronJobTestObject())
			})
			response := resource.UpdateResponse{State: state}
			r.Update(context.Background(), resource.UpdateRequest{
				State: state, Plan: tfsdk.Plan{Schema: state.Schema, Raw: plan.Raw},
			}, &response)
			if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) {
				t.Fatalf("%s failure lost state or missing error: %v", failure, response.Diagnostics)
			}
		})
	}
}

func TestCronJobPreservesZeroHistoryAndTimezoneRemoval(t *testing.T) {
	ctx := context.Background()
	actual := cronJobTestObject()
	actual.Spec.FailedJobsHistoryLimit = ptr.To(int32(0))
	actual.Spec.SuccessfulJobsHistoryLimit = ptr.To(int32(0))
	actual.Spec.TimeZone = ptr.To("Etc/UTC")
	model, _ := cronJobTestState(t, actual)
	expanded, d := expandCronJobSpec(ctx, model.Spec)
	if d.HasError() || expanded.FailedJobsHistoryLimit == nil || *expanded.FailedJobsHistoryLimit != 0 ||
		expanded.SuccessfulJobsHistoryLimit == nil || *expanded.SuccessfulJobsHistoryLimit != 0 ||
		expanded.TimeZone == nil || *expanded.TimeZone != "Etc/UTC" {
		t.Fatalf("explicit zeros/timezone lost: %#v, %v", expanded, d)
	}
	actual.Spec.TimeZone = nil
	value, d := flattenCronJobSpec(ctx, actual.Spec, model.Spec)
	if d.HasError() {
		t.Fatal(d)
	}
	raw, d := legacyValue(ctx, value)
	if d.HasError() || raw.([]interface{})[0].(map[string]interface{})["timezone"] != "" {
		t.Fatalf("removed timezone must return legacy empty string: %#v, %v", raw, d)
	}
	if strings.Contains(value.String(), "<unknown>") {
		t.Fatal("read returned unknown state")
	}
}
