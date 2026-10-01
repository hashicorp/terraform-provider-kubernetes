// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestDeploymentV1EmptyReplicasPlan(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config types.String
		prior  types.String
		want   types.String
	}{
		{name: "delegated", config: types.StringValue(""), prior: types.StringValue("2"), want: types.StringValue("2")},
		{name: "explicit-zero", config: types.StringValue("0"), prior: types.StringValue("2"), want: types.StringValue("0")},
		{name: "unknown-config", config: types.StringUnknown(), prior: types.StringValue("2"), want: types.StringUnknown()},
		{name: "omitted-config", config: types.StringNull(), prior: types.StringValue("2"), want: types.StringNull()},
		{name: "create", config: types.StringValue(""), prior: types.StringNull(), want: types.StringValue("")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := planmodifier.StringResponse{PlanValue: tc.config}
			deploymentEmptyReplicasUseState{}.PlanModifyString(context.Background(), planmodifier.StringRequest{
				ConfigValue: tc.config, PlanValue: tc.config, StateValue: tc.prior,
			}, &response)
			if response.Diagnostics.HasError() || !response.PlanValue.Equal(tc.want) {
				t.Fatalf("planned replicas = %s, want %s; diagnostics %v", response.PlanValue, tc.want, response.Diagnostics)
			}
		})
	}
}

func TestDeploymentV1HistoricalStatePreservesConfiguredFields(t *testing.T) {
	ctx := context.Background()
	deployment := &DeploymentV1{}
	for _, version := range []int64{0, 1} {
		t.Run(string(rune('0'+version)), func(t *testing.T) {
			input := deploymentHistoricalStateFixture(version)
			encoded, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			actual, diagnostics := deployment.decodeHistoricalState(ctx, &tfprotov6.RawState{JSON: encoded}, version)
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			expectedJSON, err := json.Marshal(deploymentHistoricalStateFixture(1))
			if err != nil {
				t.Fatal(err)
			}
			expected, err := (&tfprotov6.RawState{JSON: expectedJSON}).Unmarshal(actual.Schema.Type().TerraformType(ctx))
			if err != nil {
				t.Fatal(err)
			}
			if !expected.Equal(actual.Raw) {
				t.Fatal("historical conversion lost or changed configured state before Read")
			}
			if version == 0 {
				response := resource.UpgradeStateResponse{State: tfsdk.State{Schema: actual.Schema}}
				deployment.UpgradeState(ctx)[0].StateUpgrader(ctx, resource.UpgradeStateRequest{RawState: &tfprotov6.RawState{JSON: encoded}}, &response)
				if response.Diagnostics.HasError() || !response.State.Raw.Equal(expected) {
					t.Fatalf("registered schema-v0 upgrader did not preserve complete state: %v", response.Diagnostics)
				}
			}
		})
	}
}

func deploymentHistoricalStateFixture(version int64) map[string]any {
	resources := func(cpu string) []any {
		limits := any(map[string]any{"cpu": cpu, "memory": "64Mi"})
		requests := any(map[string]any{"cpu": "100m", "memory": "32Mi"})
		if version == 0 {
			limits = []any{limits}
			requests = []any{requests}
		}
		return []any{map[string]any{"limits": limits, "requests": requests}}
	}
	return map[string]any{
		"id": "default/historical", "wait_for_rollout": false,
		"timeouts": map[string]any{"create": "10m", "update": "5m", "delete": "7m"},
		"metadata": []any{map[string]any{
			"name": "historical", "namespace": "default", "uid": "existing-uid", "generation": 7,
			"resource_version": "42", "generate_name": "", "labels": map[string]any{"app": "historical"},
		}},
		"spec": []any{map[string]any{
			"replicas": "2", "min_ready_seconds": 3, "progress_deadline_seconds": 120, "revision_history_limit": 2, "paused": false,
			"selector": []any{map[string]any{"match_labels": map[string]any{"app": "historical"}}},
			"strategy": []any{map[string]any{"type": "RollingUpdate", "rolling_update": []any{map[string]any{"max_surge": "50%", "max_unavailable": "0"}}}},
			"template": []any{map[string]any{
				"metadata": []any{map[string]any{"labels": map[string]any{"app": "historical"}, "annotations": map[string]any{"example.com/key": "value"}}},
				"spec": []any{map[string]any{
					"container":      []any{map[string]any{"name": "main", "image": "busybox:1.36", "resources": resources("250m")}},
					"init_container": []any{map[string]any{"name": "init", "image": "busybox:1.36", "resources": resources("500m")}},
				}},
			}},
		}},
	}
}

type deploymentReadClientsets struct {
	kubernetes.KubeClientsets
	client *k8sclient.Clientset
}

func (clients deploymentReadClientsets) MainClientset() (*k8sclient.Clientset, error) {
	return clients.client, nil
}
func (deploymentReadClientsets) GetIgnoreAnnotations() []string { return nil }
func (deploymentReadClientsets) GetIgnoreLabels() []string      { return nil }

func TestDeploymentV1ReadErrorPreservesState(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		reason string
	}{
		{name: "forbidden", status: http.StatusForbidden, reason: "Forbidden"},
		{name: "transient", status: http.StatusServiceUnavailable, reason: "ServiceUnavailable"},
		{name: "not-found", status: http.StatusNotFound, reason: "NotFound"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != "/apis/apps/v1/namespaces/default/deployments/historical" {
					t.Errorf("unexpected API request: %s %s", req.Method, req.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_ = json.NewEncoder(w).Encode(map[string]any{"kind": "Status", "apiVersion": "v1", "status": "Failure", "reason": tc.reason, "code": tc.status})
			}))
			defer server.Close()
			client, err := k8sclient.NewForConfig(&rest.Config{Host: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			deployment := &DeploymentV1{SDKv2Meta: func() any { return deploymentReadClientsets{client: client} }}
			encoded, err := json.Marshal(deploymentHistoricalStateFixture(1))
			if err != nil {
				t.Fatal(err)
			}
			state, diagnostics := deployment.decodeHistoricalState(context.Background(), &tfprotov6.RawState{JSON: encoded}, 1)
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			response := resource.ReadResponse{State: state}
			deployment.Read(context.Background(), resource.ReadRequest{State: state}, &response)
			if tc.status == http.StatusNotFound {
				if response.Diagnostics.HasError() || !response.State.Raw.IsNull() {
					t.Fatalf("confirmed absence must remove state, diagnostics: %v", response.Diagnostics)
				}
			} else if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) {
				t.Fatalf("API errors must retain existing state with an error, diagnostics: %v", response.Diagnostics)
			}
		})
	}
}
