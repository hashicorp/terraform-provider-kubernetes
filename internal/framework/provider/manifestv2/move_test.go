// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestMoveState(t *testing.T) {
	ctx := context.Background()
	r := &ManifestV2{}
	var schema resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schema)

	for name, tc := range map[string]struct {
		sourceType, provider, raw string
		want                      identity
		manager, uid              string
		force, skipped            bool
		waits                     int
	}{
		"kubernetes_manifest": {
			sourceType: "kubernetes_manifest", provider: "registry.terraform.io/hashicorp/kubernetes",
			raw: `{"manifest":{"type":["object",{"apiVersion":"string","kind":"string","metadata":["object",{"name":"string"}],"data":["object",{"a":"string"}]}],` +
				`"value":{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm"},"data":{"a":"1"}}},` +
				`"object":{"type":["object",{"metadata":["object",{"namespace":"string"}]}],"value":{"metadata":{"namespace":"default"}}},` +
				`"computed_fields":["spec.replicas"],"field_manager":[{"name":"custom","force_conflicts":true}],` +
				`"wait":[{"rollout":null,"fields":{"status.phase":"Running"},"condition":[{"type":"Ready","status":"True"}]}],` +
				`"wait_for":null,"timeouts":[{"create":"5m","update":null,"delete":null}]}`,
			want: identity{"v1", "ConfigMap", "default", "cm"}, manager: "custom", force: true, waits: 2,
		},
		// yaml_body is taken over as it is. Client-side apply leaves no field
		// manager to keep.
		"kubectl_manifest": {
			sourceType: "kubectl_manifest", provider: "registry.example.com/acme/kubectl",
			raw: `{"yaml_body":"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: web\n  namespace: prod\n","namespace":"prod","uid":"u-2",` +
				`"server_side_apply":false,"field_manager":"kubectl","force_conflicts":null}`,
			want: identity{"apps/v1", "Deployment", "prod", "web"}, manager: "Terraform", uid: "u-2",
		},
		// The object is in the namespace the source applied it to.
		"kubectl_manifest in another namespace than its YAML": {
			sourceType: "kubectl_manifest", provider: "registry.example.com/acme/kubectl",
			raw:  `{"yaml_body":"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: c\n","namespace":"prod","uid":"u-4","server_side_apply":false,"field_manager":"kubectl"}`,
			want: identity{"v1", "ConfigMap", "prod", "c"}, manager: "Terraform", uid: "u-4",
		},
		// This resource applies as Terraform; the source's manager is kept
		// privately (see TestManifestV2MoveFromKubectlManifest).
		"kubectl_manifest with server-side apply": {
			sourceType: "kubectl_manifest", provider: "registry.example.com/acme/kubectl",
			raw:  `{"yaml_body":"apiVersion: v1\nkind: Namespace\nmetadata:\n  name: prod\n","namespace":"","uid":"u-3","server_side_apply":true,"field_manager":"kubectl"}`,
			want: identity{"v1", "Namespace", "", "prod"}, manager: "Terraform", uid: "u-3",
		},
		"kubectl_manifest with server-side apply and no field_manager": {
			sourceType: "kubectl_manifest", provider: "registry.example.com/acme/kubectl",
			raw:  `{"yaml_body":"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: web\n  namespace: prod\n","uid":"u-5","server_side_apply":true,"force_conflicts":false}`,
			want: identity{"apps/v1", "Deployment", "prod", "web"}, manager: "Terraform", uid: "u-5",
		},
		"another resource type":               {sourceType: "kubernetes_config_map", provider: "registry.terraform.io/hashicorp/kubernetes", raw: `{}`, skipped: true},
		"kubernetes_manifest of a look-alike": {sourceType: "kubernetes_manifest", provider: "registry.terraform.io/example/kubernetes", raw: `{}`, skipped: true},
	} {
		t.Run(name, func(t *testing.T) {
			req := resource.MoveStateRequest{SourceTypeName: tc.sourceType, SourceProviderAddress: tc.provider, SourceRawState: &tfprotov6.RawState{JSON: []byte(tc.raw)}}
			resp := resource.MoveStateResponse{TargetState: tfsdk.State{Schema: schema.Schema, Raw: tftypes.NewValue(schema.Schema.Type().TerraformType(ctx), nil)}}
			for _, mover := range r.MoveState(ctx) {
				mover.StateMover(ctx, req, &resp)
			}
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if tc.skipped {
				if !resp.TargetState.Raw.IsNull() {
					t.Fatal("the source must be left to other movers")
				}
				return
			}
			var got model
			if diags := resp.TargetState.Get(ctx, &got); diags.HasError() {
				t.Fatal(diags)
			}
			_, declared, known, err := got.declared()
			if err != nil || !known || declared.Name != tc.want.Name || declared.Kind != tc.want.Kind {
				t.Fatalf("manifest identifies %v (known %t, error %v)", declared, known, err)
			}
			manager, force := got.manager()
			if got.identity() != tc.want || manager != tc.manager || force != tc.force || got.UID.ValueString() != tc.uid {
				t.Fatalf("got %v, manager %q (force %t), uid %q", got.identity(), manager, force, got.UID.ValueString())
			}
			expected, err := expectations(ctx, got.Wait, declared)
			if err != nil || len(expected) != tc.waits {
				t.Fatalf("got %d wait conditions (error %v), want %d", len(expected), err, tc.waits)
			}
		})
	}
}
