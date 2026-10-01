// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestDaemonSetHistoricalStateThroughMux(t *testing.T) {
	testDaemonSetHistoricalStateThroughMux(t, false)
}

func TestDaemonSetHistoricalNullCollectionsThroughMux(t *testing.T) {
	testDaemonSetHistoricalStateThroughMux(t, true)
}

func testDaemonSetHistoricalStateThroughMux(t *testing.T, nullCollections bool) {
	t.Helper()
	ctx := context.Background()
	server, err := testAccProviderFactories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	valueType := schemas.ResourceSchemas["kubernetes_daemon_set_v1"].ValueType()
	rawState := func(version int64) *tfprotov6.RawState {
		var values map[string]interface{}
		if err := json.Unmarshal(daemonSetHistoricalStateJSON(version), &values); err != nil {
			t.Fatal(err)
		}
		if nullCollections {
			values["metadata"].([]interface{})[0].(map[string]interface{})["annotations"] = nil
			spec := values["spec"].([]interface{})[0].(map[string]interface{})
			template := spec["template"].([]interface{})[0].(map[string]interface{})
			template["metadata"].([]interface{})[0].(map[string]interface{})["annotations"] = nil
			podSpec := template["spec"].([]interface{})[0].(map[string]interface{})
			podSpec["node_selector"] = nil
			container := podSpec["container"].([]interface{})[0].(map[string]interface{})
			container["args"], container["command"] = nil, nil
		}
		populateDaemonSetHistoricalBlocks(values, schemas.ResourceSchemas["kubernetes_daemon_set_v1"].Block)
		data, err := json.Marshal(values)
		if err != nil {
			t.Fatal(err)
		}
		return &tfprotov6.RawState{JSON: data}
	}
	expected, err := rawState(1).Unmarshal(valueType)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []int64{0, 1} {
		t.Run(fmt.Sprintf("upgrade-v%d", version), func(t *testing.T) {
			response, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
				TypeName: "kubernetes_daemon_set_v1", Version: version,
				RawState: rawState(version),
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, diagnostic := range response.Diagnostics {
				if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
					t.Fatalf("%s: %s", diagnostic.Summary, diagnostic.Detail)
				}
			}
			if response.UpgradedState == nil {
				t.Fatal("no upgraded state")
			}
			actual, err := response.UpgradedState.Unmarshal(valueType)
			if err != nil {
				t.Fatal(err)
			}
			checkDaemonSetHistoricalState(t, actual, expected)
		})
		t.Run(fmt.Sprintf("alias-move-v%d", version), func(t *testing.T) {
			response, err := server.MoveResourceState(ctx, &tfprotov6.MoveResourceStateRequest{
				SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
				SourceTypeName:        "kubernetes_daemonset",
				SourceSchemaVersion:   version,
				SourceState:           rawState(version),
				TargetTypeName:        "kubernetes_daemon_set_v1",
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, diagnostic := range response.Diagnostics {
				if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
					t.Fatalf("%s: %s", diagnostic.Summary, diagnostic.Detail)
				}
			}
			if response.TargetState == nil {
				t.Fatal("no moved state")
			}
			actual, err := response.TargetState.Unmarshal(valueType)
			if err != nil {
				t.Fatal(err)
			}
			checkDaemonSetHistoricalState(t, actual, expected)
		})
	}
}

func checkDaemonSetHistoricalState(t *testing.T, actual, expected tftypes.Value) {
	t.Helper()
	diffs, err := actual.Diff(expected)
	if err != nil {
		t.Fatal(err)
	}
	for _, difference := range diffs {
		t.Errorf("state changed before Read at %s", difference.Path)
	}
}

// SDKv2 persisted omitted blocks as empty collections, including inside each
// configured ancestor. Keep this fixture realistic without hand-listing PodSpec.
func populateDaemonSetHistoricalBlocks(values map[string]interface{}, block *tfprotov6.SchemaBlock) {
	for _, nested := range block.BlockTypes {
		if nested.Nesting != tfprotov6.SchemaNestedBlockNestingModeList && nested.Nesting != tfprotov6.SchemaNestedBlockNestingModeSet {
			continue
		}
		if values[nested.TypeName] == nil {
			values[nested.TypeName] = []interface{}{}
		}
		for _, entry := range values[nested.TypeName].([]interface{}) {
			populateDaemonSetHistoricalBlocks(entry.(map[string]interface{}), nested.Block)
		}
	}
}

func daemonSetHistoricalStateJSON(version int64) []byte {
	requests := `{"cpu":"100m","memory":"32Mi"}`
	limits := `{"cpu":"0.5","memory":"64Mi"}`
	if version == 0 {
		requests = "[" + requests + "]"
		limits = "[" + limits + "]"
	}
	resources := fmt.Sprintf(`[{"requests":%s,"limits":%s}]`, requests, limits)
	return fmt.Appendf(nil, `{
  "id":"default/historical",
  "wait_for_rollout":false,
  "metadata":[{
    "name":"historical","namespace":"default","uid":"preserved-uid",
    "generation":3,"resource_version":"123",
    "labels":{"app":"historical"},"annotations":{"example.com/owner":"kept"}
  }],
  "spec":[{
    "min_ready_seconds":2,"revision_history_limit":4,
    "selector":[{"match_labels":{"app":"historical"}}],
    "strategy":[{"type":"OnDelete","rolling_update":[]}],
    "template":[{
      "metadata":[{"labels":{"app":"historical"},"annotations":{}}],
      "spec":[{
        "container":[{"name":"main","image":"busybox:1.36","args":["3600"],"resources":%s}],
        "init_container":[{"name":"prepare","image":"busybox:1.36","command":["true"],"resources":%s}],
        "image_pull_secrets":[{"name":"registry"}],
        "node_selector":{"kubernetes.io/os":"linux"}
      }]
    }]
  }],
  "timeouts":{"create":"20m","update":"21m","delete":"22m"}
}`, resources, resources)
}
