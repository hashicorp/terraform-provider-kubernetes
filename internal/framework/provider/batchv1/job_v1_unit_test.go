// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/batchv1"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	batchapi "k8s.io/api/batch/v1"
	coreapi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

type jobUnitClientsets struct {
	kubernetes.KubeClientsets
	client *k8sclient.Clientset
	err    error
}

func (c jobUnitClientsets) MainClientset() (*k8sclient.Clientset, error) { return c.client, c.err }
func (jobUnitClientsets) GetIgnoreAnnotations() []string                 { return []string{"ignored.example/.*"} }
func (jobUnitClientsets) GetIgnoreLabels() []string                      { return []string{"ignored.example/.*"} }

const jobUnitStateJSON = `{
	"id":"default/test","wait_for_completion":false,"timeouts":null,
	"metadata":[{"name":"test","namespace":"default","generate_name":null,
		"annotations":null,"labels":null,"generation":1,"uid":"uid-1","resource_version":"1"}],
	"spec":[{"backoff_limit":6,"completions":1,"completion_mode":"NonIndexed",
		"manual_selector":false,"parallelism":1,"backoff_limit_per_index":0,"max_failed_indexes":0,
		"template":[{"metadata":[{"name":"","generate_name":null,"annotations":null,"labels":null}],
			"spec":[{"restart_policy":"Never","automount_service_account_token":true,
				"dns_policy":"ClusterFirst","enable_service_links":true,"host_ipc":false,
				"host_network":false,"host_pid":false,"share_process_namespace":false,
				"termination_grace_period_seconds":30,
				"container":[{"name":"hello","image":"busybox:1.36","stdin":false,
					"stdin_once":false,"tty":false,"termination_message_path":"/dev/termination-log","command":["true"]}]}]}]}]
}`

func jobUnitState(t *testing.T, r *batchv1.JobV1, input string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	var response fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	raw := tfprotov6.RawState{JSON: []byte(input)}
	value, err := raw.Unmarshal(response.Schema.Type().TerraformType(ctx))
	if err != nil {
		t.Fatal(err)
	}
	return tfsdk.State{Schema: response.Schema, Raw: value}
}

func jobUnitModel(t *testing.T, state tfsdk.State) batchv1.JobV1Model {
	t.Helper()
	var model batchv1.JobV1Model
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func jobUnitModelState(t *testing.T, state tfsdk.State, model batchv1.JobV1Model) tfsdk.State {
	t.Helper()
	if diags := state.Set(context.Background(), model); diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

func jobUnitPlannedState(t *testing.T, state tfsdk.State) tfsdk.State {
	t.Helper()
	var prepareObject func(tftypes.Value, map[string]schema.Attribute, map[string]schema.Block) tftypes.Value
	var prepareList func(tftypes.Value, map[string]schema.Attribute, map[string]schema.Block) tftypes.Value
	prepareList = func(value tftypes.Value, attributes map[string]schema.Attribute, blocks map[string]schema.Block) tftypes.Value {
		if value.IsNull() || !value.IsKnown() {
			return value
		}
		var elements []tftypes.Value
		if err := value.As(&elements); err != nil {
			t.Fatal(err)
		}
		for i := range elements {
			elements[i] = prepareObject(elements[i], attributes, blocks)
		}
		return tftypes.NewValue(value.Type(), elements)
	}
	prepareObject = func(value tftypes.Value, attributes map[string]schema.Attribute, blocks map[string]schema.Block) tftypes.Value {
		var fields map[string]tftypes.Value
		if err := value.As(&fields); err != nil {
			t.Fatal(err)
		}
		for name, attribute := range attributes {
			child := fields[name]
			if attribute.IsComputed() && (!attribute.IsOptional() || child.IsNull()) {
				fields[name] = tftypes.NewValue(child.Type(), tftypes.UnknownValue)
				continue
			}
			if nested, ok := attribute.(schema.ListNestedAttribute); ok {
				fields[name] = prepareList(child, nested.NestedObject.Attributes, nil)
			}
		}
		for name, block := range blocks {
			if nested, ok := block.(schema.ListNestedBlock); ok {
				fields[name] = prepareList(fields[name], nested.NestedObject.Attributes, nested.NestedObject.Blocks)
			}
		}
		return tftypes.NewValue(value.Type(), fields)
	}
	resourceSchema, ok := state.Schema.(schema.Schema)
	if !ok {
		t.Fatalf("unexpected resource schema %T", state.Schema)
	}
	state.Raw = prepareObject(state.Raw, resourceSchema.Attributes, resourceSchema.Blocks)
	return state
}

func jobUnitAPIJob() *batchapi.Job {
	return &batchapi.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default", UID: "uid-1", ResourceVersion: "2"},
		Spec: batchapi.JobSpec{
			BackoffLimit: ptr.To(int32(6)), Completions: ptr.To(int32(1)),
			CompletionMode: ptr.To(batchapi.NonIndexedCompletion), Parallelism: ptr.To(int32(1)),
			ManualSelector: ptr.To(false),
			Template: coreapi.PodTemplateSpec{Spec: coreapi.PodSpec{
				AutomountServiceAccountToken: ptr.To(true), DNSPolicy: coreapi.DNSClusterFirst,
				EnableServiceLinks: ptr.To(true), ShareProcessNamespace: ptr.To(false),
				TerminationGracePeriodSeconds: ptr.To(int64(30)), RestartPolicy: coreapi.RestartPolicyNever,
				Containers: []coreapi.Container{{
					Name: "hello", Image: "busybox:1.36", Command: []string{"true"},
					TerminationMessagePath: "/dev/termination-log",
				}},
			}},
		},
	}
}

func jobUnitSpec(model *batchv1.JobV1Model, key string, value attr.Value) {
	object := model.Spec.Elements()[0].(types.Object)
	fields := object.Attributes()
	fields[key] = value
	model.Spec = types.ListValueMust(object.Type(context.Background()), []attr.Value{
		types.ObjectValueMust(object.AttributeTypes(context.Background()), fields),
	})
}

func jobUnitResource(t *testing.T, handler http.HandlerFunc) *batchv1.JobV1 {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := k8sclient.NewForConfig(&rest.Config{
		Host:          server.URL,
		ContentConfig: rest.ContentConfig{ContentType: "application/json", AcceptContentTypes: "application/json"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &batchv1.JobV1{SDKv2Meta: func() any { return jobUnitClientsets{client: client} }}
}

func jobUnitWrite(t *testing.T, w http.ResponseWriter, job *batchapi.Job) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(job); err != nil {
		t.Error(err)
	}
}

func TestJobCreateLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		wait, generated       bool
		condition             batchapi.JobConditionType
		postStatus, getStatus int
		timeout               string
		wantError             bool
	}{
		{name: "without waiting"},
		{name: "complete", wait: true, condition: batchapi.JobComplete},
		{name: "generated name", wait: true, generated: true, condition: batchapi.JobComplete},
		{name: "failed retains state", wait: true, condition: batchapi.JobFailed, wantError: true},
		{name: "disappeared while waiting", wait: true, getStatus: http.StatusNotFound},
		{name: "wait API failure retains state", wait: true, getStatus: http.StatusForbidden, wantError: true},
		{name: "wait timeout retains state", wait: true, timeout: "10ms", wantError: true},
		{name: "create failure", postStatus: http.StatusForbidden, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var created batchapi.Job
			gets := 0
			r := jobUnitResource(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodPost {
					if req.URL.Path != "/apis/batch/v1/namespaces/default/jobs" {
						t.Errorf("unexpected create path %s", req.URL.Path)
					}
					if tc.postStatus != 0 {
						http.Error(w, "forbidden", tc.postStatus)
						return
					}
					if err := json.NewDecoder(req.Body).Decode(&created); err != nil {
						t.Error(err)
					}
					if created.Spec.Template.Spec.RestartPolicy != coreapi.RestartPolicyNever {
						t.Errorf("restart policy = %q", created.Spec.Template.Spec.RestartPolicy)
					}
					if tc.generated && (created.Name != "" || created.GenerateName != "prefix-") {
						t.Errorf("generated-name request: %+v", created.ObjectMeta)
					}
					created.Name, created.Namespace = "test", "default"
					if tc.generated {
						created.Name = "prefix-generated"
					}
					created.UID, created.ResourceVersion = "uid-created", "2"
					created.Labels = map[string]string{"batch.kubernetes.io/job-name": created.Name}
					jobUnitWrite(t, w, &created)
					return
				}
				gets++
				if tc.getStatus != 0 {
					http.Error(w, "lookup failed", tc.getStatus)
					return
				}
				if tc.condition != "" {
					created.Status.Conditions = []batchapi.JobCondition{{Type: tc.condition, Status: coreapi.ConditionTrue}}
				}
				jobUnitWrite(t, w, &created)
			})
			state := jobUnitState(t, r, jobUnitStateJSON)
			model := jobUnitModel(t, state)
			model.ID = types.StringNull()
			model.WaitForCompletion = types.BoolValue(tc.wait)
			if tc.generated {
				model.Metadata[0].Name = types.StringUnknown()
				model.Metadata[0].GenerateName = types.StringValue("prefix-")
			}
			if tc.timeout != "" {
				model.Timeouts.Object = types.ObjectValueMust(model.Timeouts.AttributeTypes(context.Background()), map[string]attr.Value{
					"create": types.StringValue(tc.timeout), "update": types.StringNull(), "delete": types.StringNull(),
				})
			}
			planned := jobUnitPlannedState(t, jobUnitModelState(t, state, model))
			response := fwresource.CreateResponse{
				State:    tfsdk.State{Schema: state.Schema, Raw: tftypes.NewValue(state.Raw.Type(), nil)},
				Identity: &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()},
			}
			r.Create(context.Background(), fwresource.CreateRequest{Plan: tfsdk.Plan{Schema: state.Schema, Raw: planned.Raw}}, &response)
			if response.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics = %v, want error %t", response.Diagnostics, tc.wantError)
			}
			if tc.postStatus != 0 {
				if !response.State.Raw.IsNull() {
					t.Fatal("failed create wrote state")
				}
				return
			}
			got := jobUnitModel(t, response.State)
			if got.ID.ValueString() != "default/"+created.Name || got.Metadata[0].UID.ValueString() != "uid-created" {
				t.Fatalf("created identity not retained: %+v", got)
			}
			var identity common.NamespacedResourceIdentity
			if diags := response.Identity.Get(context.Background(), &identity); diags.HasError() {
				t.Fatal(diags)
			}
			if identity.Name.ValueString() != created.Name || identity.Namespace.ValueString() != "default" || identity.Kind.ValueString() != "Job" {
				t.Fatalf("unexpected identity: %+v", identity)
			}
			if !tc.wait && gets != 0 {
				t.Fatalf("wait disabled but made %d GETs", gets)
			}
		})
	}
}

func TestJobReadFilteringAndDisappearance(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		manual bool
	}{
		{name: "empty selector is safe"},
		{name: "manual selector labels", manual: true},
		{name: "not found", status: http.StatusNotFound},
		{name: "forbidden", status: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := jobUnitResource(t, func(w http.ResponseWriter, req *http.Request) {
				if tc.status != 0 {
					http.Error(w, "lookup failed", tc.status)
					return
				}
				job := batchapi.Job{
					ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default", UID: "uid-1",
						Labels:      map[string]string{"managed": "yes", "batch.kubernetes.io/job-name": "test", "ignored.example/key": "external"},
						Annotations: map[string]string{"ignored.example/key": "external"},
					},
					Spec: batchapi.JobSpec{ManualSelector: ptr.To(tc.manual)},
				}
				job.Spec.Template.Labels = map[string]string{"controller-uid": "generated"}
				jobUnitWrite(t, w, &job)
			})
			state := jobUnitState(t, r, jobUnitStateJSON)
			if tc.manual {
				model := jobUnitModel(t, state)
				model.Metadata[0].Labels = types.MapValueMust(types.StringType, map[string]attr.Value{
					"batch.kubernetes.io/job-name": types.StringValue("test"),
				})
				state = jobUnitModelState(t, state, model)
			}
			response := fwresource.ReadResponse{State: state}
			r.Read(context.Background(), fwresource.ReadRequest{State: state}, &response)
			if tc.status == http.StatusNotFound {
				if !response.State.Raw.IsNull() || response.Diagnostics.HasError() {
					t.Fatalf("not-found response: %+v", response)
				}
				return
			}
			if tc.status == http.StatusForbidden {
				if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) {
					t.Fatal("API error must preserve prior state")
				}
				return
			}
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			got := jobUnitModel(t, response.State)
			labels := got.Metadata[0].Labels.Elements()
			expectedLabels := 1
			if tc.manual {
				expectedLabels = 2
				if labels["batch.kubernetes.io/job-name"] != types.StringValue("test") {
					t.Fatalf("manual selector lost its managed job label: %v", labels)
				}
			}
			if len(labels) != expectedLabels || labels["managed"] != types.StringValue("yes") {
				t.Fatalf("unexpected filtered labels: %v", labels)
			}
			if !got.Metadata[0].Annotations.IsNull() {
				t.Fatalf("unexpected annotations: %s", got.Metadata[0].Annotations)
			}
		})
	}
}

func TestJobUpdateMutableFields(t *testing.T) {
	for _, tc := range []struct {
		field, apiName string
		old, new       attr.Value
		expected       any
	}{
		{"ttl_seconds_after_finished", "ttlSecondsAfterFinished", types.StringNull(), types.StringValue("0"), float64(0)},
		{"ttl_seconds_after_finished", "ttlSecondsAfterFinished", types.StringValue("90"), types.StringNull(), nil},
		{"max_failed_indexes", "maxFailedIndexes", types.Int64Null(), types.Int64Value(0), float64(0)},
		{"max_failed_indexes", "maxFailedIndexes", types.Int64Value(0), types.Int64Value(1), float64(1)},
		{"max_failed_indexes", "maxFailedIndexes", types.Int64Value(3), types.Int64Value(0), float64(0)},
		{"active_deadline_seconds", "activeDeadlineSeconds", types.Int64Null(), types.Int64Value(30), float64(30)},
		{"active_deadline_seconds", "activeDeadlineSeconds", types.Int64Value(30), types.Int64Null(), nil},
		{"backoff_limit", "backoffLimit", types.Int64Value(6), types.Int64Value(0), float64(0)},
		{"parallelism", "parallelism", types.Int64Value(1), types.Int64Value(0), float64(0)},
		{"manual_selector", "manualSelector", types.BoolValue(false), types.BoolValue(true), true},
	} {
		t.Run(tc.field+"/"+tc.new.String(), func(t *testing.T) {
			var patches []map[string]any
			r := jobUnitResource(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodPatch || req.Header.Get("Content-Type") != "application/json-patch+json" {
					t.Errorf("unexpected update %s %s", req.Method, req.Header.Get("Content-Type"))
				}
				if err := json.NewDecoder(req.Body).Decode(&patches); err != nil {
					t.Error(err)
				}
				job := jobUnitAPIJob()
				if tc.field == "max_failed_indexes" {
					job.Spec.CompletionMode = ptr.To(batchapi.IndexedCompletion)
					job.Spec.BackoffLimitPerIndex = ptr.To(int32(0))
				}
				encoded, err := json.Marshal(job)
				if err != nil {
					t.Error(err)
					return
				}
				var document map[string]any
				if err := json.Unmarshal(encoded, &document); err != nil {
					t.Error(err)
					return
				}
				for _, patch := range patches {
					document["spec"].(map[string]any)[strings.TrimPrefix(patch["path"].(string), "/spec/")] = patch["value"]
				}
				encoded, err = json.Marshal(document)
				if err != nil {
					t.Error(err)
					return
				}
				if err := json.Unmarshal(encoded, job); err != nil {
					t.Error(err)
					return
				}
				jobUnitWrite(t, w, job)
			})
			state := jobUnitState(t, r, jobUnitStateJSON)
			model := jobUnitModel(t, state)
			if tc.field == "max_failed_indexes" {
				jobUnitSpec(&model, "completion_mode", types.StringValue("Indexed"))
			}
			jobUnitSpec(&model, tc.field, tc.old)
			state = jobUnitModelState(t, state, model)
			jobUnitSpec(&model, tc.field, tc.new)
			plan := jobUnitPlannedState(t, jobUnitModelState(t, state, model))
			response := fwresource.UpdateResponse{State: state}
			r.Update(context.Background(), fwresource.UpdateRequest{State: state, Plan: tfsdk.Plan{Schema: state.Schema, Raw: plan.Raw}}, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			if len(patches) != 1 || patches[0]["path"] != "/spec/"+tc.apiName || patches[0]["value"] != tc.expected {
				t.Fatalf("patches = %#v, expected %s=%v", patches, tc.apiName, tc.expected)
			}
			if got := jobUnitModel(t, response.State); got.ID.ValueString() != "default/test" {
				t.Fatalf("update lost ID: %s", got.ID)
			}
		})
	}
}

func TestJobDeleteLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		deleteStatus, readStatus int
		timeout                  bool
		wantError                bool
	}{
		{"foreground deletion", 0, http.StatusNotFound, false, false},
		{"already absent", http.StatusNotFound, 0, false, false},
		{"delete error", http.StatusForbidden, 0, false, true},
		{"poll error", 0, http.StatusForbidden, false, true},
		{"poll timeout", 0, 0, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := jobUnitResource(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodDelete {
					var options metav1.DeleteOptions
					if err := json.NewDecoder(req.Body).Decode(&options); err != nil {
						t.Error(err)
					}
					if options.PropagationPolicy == nil || *options.PropagationPolicy != metav1.DeletePropagationForeground {
						t.Errorf("incorrect delete propagation: %+v", options)
					}
					if tc.deleteStatus != 0 {
						http.Error(w, "delete failed", tc.deleteStatus)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","status":"Success"}`)
					return
				}
				if tc.readStatus != 0 {
					http.Error(w, "get failed", tc.readStatus)
					return
				}
				jobUnitWrite(t, w, &batchapi.Job{ObjectMeta: metav1.ObjectMeta{Name: "test"}})
			})
			raw := jobUnitStateJSON
			if tc.timeout {
				raw = strings.Replace(raw, `"timeouts":null`, `"timeouts":{"delete":"10ms"}`, 1)
			}
			state := jobUnitState(t, r, raw)
			response := fwresource.DeleteResponse{State: state}
			r.Delete(context.Background(), fwresource.DeleteRequest{State: state}, &response)
			if response.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics = %v, expected error %t", response.Diagnostics, tc.wantError)
			}
		})
	}
}

func TestJobRepresentationOnlyUpdateDoesNotWrite(t *testing.T) {
	requests := 0
	r := jobUnitResource(t, func(w http.ResponseWriter, req *http.Request) {
		requests++
		if req.Method != http.MethodGet {
			t.Errorf("representation-only update must not write to Kubernetes: %s", req.Method)
		}
		jobUnitWrite(t, w, jobUnitAPIJob())
	})
	state := jobUnitState(t, r, jobUnitStateJSON)
	model := jobUnitModel(t, state)
	jobUnitSpec(&model, "active_deadline_seconds", types.Int64Value(0))
	state = jobUnitModelState(t, state, model)
	jobUnitSpec(&model, "active_deadline_seconds", types.Int64Null())
	plan := jobUnitPlannedState(t, jobUnitModelState(t, state, model))
	response := fwresource.UpdateResponse{State: state}
	r.Update(context.Background(), fwresource.UpdateRequest{
		State: state, Plan: tfsdk.Plan{Schema: state.Schema, Raw: plan.Raw},
	}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if requests != 1 {
		t.Fatalf("unexpected request count %d", requests)
	}
}

func TestJobUpdatePreservesUnmanagedMetadata(t *testing.T) {
	for _, tc := range []struct {
		name           string
		prior, planned map[string]attr.Value
	}{
		{"first managed key", nil, map[string]attr.Value{"managed": types.StringValue("new")}},
		{"remove managed key", map[string]attr.Value{"managed": types.StringValue("old")}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			job := jobUnitAPIJob()
			job.Labels = map[string]string{"external": "keep", "ignored.example/key": "keep"}
			job.Annotations = map[string]string{"external": "keep", "ignored.example/key": "keep"}
			for key, value := range tc.prior {
				job.Labels[key] = value.(types.String).ValueString()
				job.Annotations[key] = value.(types.String).ValueString()
			}
			r := jobUnitResource(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodPatch {
					var operations []map[string]any
					if err := json.NewDecoder(req.Body).Decode(&operations); err != nil {
						t.Error(err)
					}
					for _, operation := range operations {
						pointer := operation["path"].(string)
						for _, field := range []struct {
							name   string
							values map[string]string
						}{
							{"labels", job.Labels}, {"annotations", job.Annotations},
						} {
							if pointer == "/metadata/"+field.name {
								t.Errorf("metadata patch replaced entire %s map: %#v", field.name, operation)
								continue
							}
							prefix := "/metadata/" + field.name + "/"
							if strings.HasPrefix(pointer, prefix) {
								key := strings.TrimPrefix(pointer, prefix)
								if operation["op"] == "remove" {
									delete(field.values, key)
								} else {
									field.values[key] = operation["value"].(string)
								}
							}
						}
					}
				}
				jobUnitWrite(t, w, job)
			})
			state := jobUnitState(t, r, jobUnitStateJSON)
			model := jobUnitModel(t, state)
			model.Metadata[0].Labels = types.MapValueMust(types.StringType, tc.prior)
			model.Metadata[0].Annotations = types.MapValueMust(types.StringType, tc.prior)
			state = jobUnitModelState(t, state, model)
			model.Metadata[0].Labels = types.MapValueMust(types.StringType, tc.planned)
			model.Metadata[0].Annotations = types.MapValueMust(types.StringType, tc.planned)
			plan := jobUnitPlannedState(t, jobUnitModelState(t, state, model))
			response := fwresource.UpdateResponse{State: state}
			r.Update(context.Background(), fwresource.UpdateRequest{
				State: state, Plan: tfsdk.Plan{Schema: state.Schema, Raw: plan.Raw},
			}, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			for _, values := range []map[string]string{job.Labels, job.Annotations} {
				if values["external"] != "keep" || values["ignored.example/key"] != "keep" {
					t.Fatalf("unmanaged metadata lost: %#v", values)
				}
			}
		})
	}
}

func TestJobSpecOnlyUpdateDoesNotPatchMetadata(t *testing.T) {
	job := jobUnitAPIJob()
	job.Labels = map[string]string{"managed": "keep", "external": "keep"}
	job.Annotations = map[string]string{"ignored.example/key": "keep"}
	r := jobUnitResource(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPatch {
			t.Errorf("spec-only update should make one patch, got %s", req.Method)
		}
		var operations []map[string]any
		if err := json.NewDecoder(req.Body).Decode(&operations); err != nil {
			t.Error(err)
		}
		if len(operations) != 1 || operations[0]["path"] != "/spec/parallelism" {
			t.Errorf("spec-only update touched metadata: %#v", operations)
		}
		job.Spec.Parallelism = ptr.To(int32(2))
		jobUnitWrite(t, w, job)
	})
	state := jobUnitState(t, r, jobUnitStateJSON)
	model := jobUnitModel(t, state)
	model.Metadata[0].Labels = types.MapValueMust(types.StringType, map[string]attr.Value{"managed": types.StringValue("keep")})
	state = jobUnitModelState(t, state, model)
	model.Metadata[0].Labels = types.MapUnknown(types.StringType)
	jobUnitSpec(&model, "parallelism", types.Int64Value(2))
	plan := jobUnitPlannedState(t, jobUnitModelState(t, state, model))
	response := fwresource.UpdateResponse{State: state}
	r.Update(context.Background(), fwresource.UpdateRequest{
		State: state, Plan: tfsdk.Plan{Schema: state.Schema, Raw: plan.Raw},
	}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
}

func TestJobImportAndProviderValidation(t *testing.T) {
	ctx := context.Background()
	r := &batchv1.JobV1{}
	for _, input := range []any{nil, "bad data", func() any { return "bad result" }} {
		var response fwresource.ConfigureResponse
		r.Configure(ctx, fwresource.ConfigureRequest{ProviderData: input}, &response)
		if input != nil {
			if _, ok := input.(string); ok && !response.Diagnostics.HasError() {
				t.Fatal("unexpected provider data accepted")
			}
		}
	}
	state := jobUnitState(t, r, jobUnitStateJSON)
	for _, id := range []string{"name-only", "/name", "default/", "a/b/c"} {
		response := fwresource.ImportStateResponse{State: state}
		r.ImportState(ctx, fwresource.ImportStateRequest{ID: id}, &response)
		if !response.Diagnostics.HasError() {
			t.Fatalf("invalid import ID %q accepted", id)
		}
	}
	nullIdentity := &tfsdk.ResourceIdentity{
		Schema: common.NamespacedIdentitySchema(),
		Raw:    tftypes.NewValue(common.NamespacedIdentitySchema().Type().TerraformType(ctx), nil),
	}
	idResponse := fwresource.ImportStateResponse{
		State:    state,
		Identity: &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()},
	}
	r.ImportState(ctx, fwresource.ImportStateRequest{ID: "default/imported-by-id", Identity: nullIdentity}, &idResponse)
	if idResponse.Diagnostics.HasError() {
		t.Fatal(idResponse.Diagnostics)
	}
	var importedID string
	if diags := idResponse.State.GetAttribute(ctx, path.Root("id"), &importedID); diags.HasError() || importedID != "default/imported-by-id" {
		t.Fatalf("null-identity ID import failed: %s, %v", importedID, diags)
	}
	identity := &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()}
	if diags := identity.Set(ctx, common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{Name: types.StringValue("imported"), Kind: types.StringValue("Job"), APIVersion: types.StringValue("batch/v1")},
		Namespace:        types.StringNull(),
	}); diags.HasError() {
		t.Fatal(diags)
	}
	response := fwresource.ImportStateResponse{State: state, Identity: &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()}}
	r.ImportState(ctx, fwresource.ImportStateRequest{Identity: identity}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var id string
	if diags := response.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() || id != "default/imported" {
		t.Fatalf("identity import ID = %s, diagnostics %v", id, diags)
	}
	for _, resource := range []*batchv1.JobV1{
		{},
		{SDKv2Meta: func() any { return nil }},
		{SDKv2Meta: func() any { return jobUnitClientsets{err: fmt.Errorf("client unavailable")} }},
	} {
		response := fwresource.ReadResponse{State: state}
		resource.Read(ctx, fwresource.ReadRequest{State: state}, &response)
		if !response.Diagnostics.HasError() {
			t.Fatal("invalid client configuration did not fail")
		}
	}
}

func TestJobTimeoutDefaults(t *testing.T) {
	r := &batchv1.JobV1{}
	model := jobUnitModel(t, jobUnitState(t, r, jobUnitStateJSON))
	for _, timeout := range []func(context.Context, time.Duration) (time.Duration, diag.Diagnostics){
		model.Timeouts.Create, model.Timeouts.Update, model.Timeouts.Delete,
	} {
		duration, diags := timeout(context.Background(), time.Minute)
		if diags.HasError() || duration != time.Minute {
			t.Fatalf("default timeout = %s, diagnostics %v", duration, diags)
		}
	}
}
