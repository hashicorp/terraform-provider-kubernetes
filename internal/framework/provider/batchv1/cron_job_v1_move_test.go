// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	core "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
)

func cronJobJSONValue(t *testing.T, value tftypes.Value) any {
	t.Helper()
	if value.IsNull() {
		return nil
	}
	if !value.IsKnown() {
		t.Fatal("test state must be fully known")
	}
	switch value.Type().(type) {
	case tftypes.Object, tftypes.Map:
		var values map[string]tftypes.Value
		if err := value.As(&values); err != nil {
			t.Fatal(err)
		}
		result := map[string]interface{}{}
		for key, child := range values {
			result[key] = cronJobJSONValue(t, child)
		}
		return result
	case tftypes.List, tftypes.Set:
		var values []tftypes.Value
		if err := value.As(&values); err != nil {
			t.Fatal(err)
		}
		result := make([]interface{}, len(values))
		for i, child := range values {
			result[i] = cronJobJSONValue(t, child)
		}
		return result
	default:
		switch {
		case value.Type().Is(tftypes.String):
			var result string
			if err := value.As(&result); err != nil {
				t.Fatal(err)
			}
			return result
		case value.Type().Is(tftypes.Bool):
			var result bool
			if err := value.As(&result); err != nil {
				t.Fatal(err)
			}
			return result
		case value.Type().Is(tftypes.Number):
			var result big.Float
			if err := value.As(&result); err != nil {
				t.Fatal(err)
			}
			return json.Number(result.Text('f', 0))
		}
	}
	t.Fatalf("unhandled test value %s", value.Type())
	return nil
}

func cronJobSourceState(t *testing.T, version int64) ([]byte, tfsdk.State) {
	t.Helper()
	object := cronJobTestObject()
	object.GenerateName = "test-"
	object.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Resources = core.ResourceRequirements{
		Limits: core.ResourceList{
			core.ResourceCPU: apiresource.MustParse("200m"), core.ResourceMemory: apiresource.MustParse("64Mi"),
		},
		Requests: core.ResourceList{
			core.ResourceCPU: apiresource.MustParse("100m"), core.ResourceMemory: apiresource.MustParse("32Mi"),
		},
	}
	object.Spec.JobTemplate.Spec.Template.Spec.InitContainers = []core.Container{
		object.Spec.JobTemplate.Spec.Template.Spec.Containers[0],
	}
	object.Spec.JobTemplate.Spec.Template.Spec.InitContainers[0].Name = "init"
	model, state := cronJobTestState(t, object)
	model.Metadata[0].Annotations = types.MapValueMust(types.StringType, nil)
	state = cronJobStateFromModel(t, state, model)
	source := cronJobJSONValue(t, state.Raw).(map[string]interface{})
	spec := source["spec"].([]interface{})[0].(map[string]interface{})
	delete(spec, "timezone")
	if version == 0 {
		podSpec := spec
		for _, field := range []string{"job_template", "spec", "template", "spec"} {
			podSpec = podSpec[field].([]interface{})[0].(map[string]interface{})
		}
		for _, field := range []string{"container", "init_container"} {
			for _, element := range podSpec[field].([]interface{}) {
				container := element.(map[string]interface{})
				values := container["resources"].([]interface{})[0].(map[string]interface{})
				for _, resourceMap := range []string{"limits", "requests"} {
					values[resourceMap] = []interface{}{values[resourceMap]}
				}
			}
		}
	}
	data, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	return data, state
}

func TestCronJobMoveStateAliasVersions(t *testing.T) {
	for _, version := range []int64{0, 1} {
		t.Run(string(rune('0'+version)), func(t *testing.T) {
			ctx := context.Background()
			data, expected := cronJobSourceState(t, version)
			r := &CronJobV1{SDKv2Meta: func() any {
				t.Fatal("state migration must not configure or write to Kubernetes")
				return nil
			}}
			response := resource.MoveStateResponse{
				TargetState:    tfsdk.State{Schema: expected.Schema},
				TargetIdentity: &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()},
			}
			r.MoveState(ctx)[0].StateMover(ctx, resource.MoveStateRequest{
				SourceTypeName: "kubernetes_cron_job", SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
				SourceSchemaVersion: version, SourceRawState: &tfprotov6.RawState{JSON: data},
			}, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			if !response.TargetState.Raw.Equal(expected.Raw) {
				t.Fatal("moved state lost values, list shape, or null/empty metadata")
			}
			var identity common.NamespacedResourceIdentity
			if d := response.TargetIdentity.Get(ctx, &identity); d.HasError() {
				t.Fatal(d)
			}
			if !reflect.DeepEqual(identity, cronJobIdentity("default", "test")) {
				t.Fatalf("identity must target batch/v1: %#v", identity)
			}
		})
	}
}

func TestCronJobMoveStateRejectsUnsupportedSources(t *testing.T) {
	ctx := context.Background()
	data, state := cronJobSourceState(t, 1)
	for _, test := range []struct {
		provider string
		typ      string
		version  int64
	}{
		{"registry.terraform.io/nothashicorp/kubernetes", "kubernetes_cron_job", 1},
		{"registry.terraform.io/hashicorp/kubernetes", "kubernetes_job", 1},
		{"registry.terraform.io/hashicorp/kubernetes", "kubernetes_cron_job_v1", 0},
		{"registry.terraform.io/hashicorp/kubernetes", "kubernetes_cron_job", 2},
	} {
		response := resource.MoveStateResponse{TargetState: tfsdk.State{Schema: state.Schema}}
		(&CronJobV1{}).MoveState(ctx)[0].StateMover(ctx, resource.MoveStateRequest{
			SourceProviderAddress: test.provider, SourceTypeName: test.typ,
			SourceSchemaVersion: test.version, SourceRawState: &tfprotov6.RawState{JSON: data},
		}, &response)
		if response.TargetState.Raw.Type() != nil || response.Diagnostics.HasError() {
			t.Fatalf("unsupported source was claimed: %#v", test)
		}
	}
}

func TestCronJobMoveStateOlderOptionalFields(t *testing.T) {
	ctx := context.Background()
	data, expected := cronJobSourceState(t, 0)
	var source map[string]interface{}
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	spec := source["spec"].([]interface{})[0].(map[string]interface{})
	template := spec["job_template"].([]interface{})[0].(map[string]interface{})
	jobSpec := template["spec"].([]interface{})[0].(map[string]interface{})
	for _, field := range []string{"backoff_limit_per_index", "max_failed_indexes", "completion_mode", "pod_failure_policy"} {
		delete(jobSpec, field)
	}
	data, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	response := resource.MoveStateResponse{TargetState: tfsdk.State{Schema: expected.Schema}}
	(&CronJobV1{}).MoveState(ctx)[0].StateMover(ctx, resource.MoveStateRequest{
		SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes", SourceTypeName: "kubernetes_cron_job",
		SourceSchemaVersion: 0, SourceRawState: &tfprotov6.RawState{JSON: data},
	}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if response.TargetState.Raw.IsNull() || !response.TargetState.Raw.IsFullyKnown() {
		t.Fatal("historical omitted optional fields must decode into complete known state")
	}
}

func TestCronJobMoveStateRejectsMalformedState(t *testing.T) {
	ctx := context.Background()
	data, state := cronJobSourceState(t, 1)
	for _, value := range [][]byte{
		nil, []byte(`{`), []byte(`null`), []byte(`{}`),
		[]byte(`{"id":"default/test","metadata":[],"spec":[]}`),
		[]byte(`{"id":"default/test","metadata":[{"name":"other","namespace":"default"}],"spec":[{}]}`),
	} {
		response := resource.MoveStateResponse{TargetState: tfsdk.State{Schema: state.Schema}}
		(&CronJobV1{}).MoveState(ctx)[0].StateMover(ctx, resource.MoveStateRequest{
			SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes", SourceTypeName: "kubernetes_cron_job",
			SourceSchemaVersion: 1, SourceRawState: &tfprotov6.RawState{JSON: value},
		}, &response)
		if !response.Diagnostics.HasError() {
			t.Errorf("malformed state accepted: %s", value)
		}
	}
	var source map[string]interface{}
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	source["spec"] = []interface{}{map[string]interface{}{"job_template": "invalid"}}
	broken, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	response := resource.MoveStateResponse{TargetState: tfsdk.State{Schema: state.Schema}}
	(&CronJobV1{}).MoveState(ctx)[0].StateMover(ctx, resource.MoveStateRequest{
		SourceProviderAddress: "mirror.example/hashicorp/kubernetes", SourceTypeName: "kubernetes_cron_job",
		SourceSchemaVersion: 0, SourceRawState: &tfprotov6.RawState{JSON: broken},
	}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("malformed v0 container tree must not panic or be accepted")
	}
}

func TestCronJobV1VersionZeroStateDecode(t *testing.T) {
	ctx := context.Background()
	_, state := cronJobSourceState(t, 1)
	data, err := json.Marshal(cronJobJSONValue(t, state.Raw))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := (&tfprotov6.RawState{JSON: data}).Unmarshal(state.Schema.Type().TerraformType(ctx))
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.Equal(state.Raw) {
		t.Fatal("v1 version-zero state did not retain its wire representation")
	}
}

func TestCronJobUpgradeIdentity(t *testing.T) {
	ctx := context.Background()
	for _, data := range [][]byte{nil, []byte(`{"namespace":"default","name":"test"}`)} {
		response := resource.UpgradeIdentityResponse{
			Identity: &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()},
		}
		(&CronJobV1{}).UpgradeIdentity(ctx)[0].IdentityUpgrader(ctx, resource.UpgradeIdentityRequest{
			RawIdentity: &tfprotov6.RawState{JSON: data},
		}, &response)
		if response.Diagnostics.HasError() {
			t.Fatal(response.Diagnostics)
		}
		var identity common.NamespacedResourceIdentity
		if d := response.Identity.Get(ctx, &identity); d.HasError() {
			t.Fatal(d)
		}
		if len(data) == 0 {
			if !identity.Name.IsNull() || !identity.Namespace.IsNull() || !identity.Kind.IsNull() || !identity.APIVersion.IsNull() {
				t.Fatal("pre-identity state must stay all-null until Read")
			}
		} else if !reflect.DeepEqual(identity, cronJobIdentity("default", "test")) {
			t.Fatalf("unexpected identity: %#v", identity)
		}
	}
}
