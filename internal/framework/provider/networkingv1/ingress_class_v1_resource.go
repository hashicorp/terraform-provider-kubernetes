// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

var (
	_ resource.Resource                    = (*IngressClassV1)(nil)
	_ resource.ResourceWithConfigure       = (*IngressClassV1)(nil)
	_ resource.ResourceWithImportState     = (*IngressClassV1)(nil)
	_ resource.ResourceWithIdentity        = (*IngressClassV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*IngressClassV1)(nil)
	_ resource.ResourceWithMoveState       = (*IngressClassV1)(nil)
)

type IngressClassV1 struct {
	SDKv2Meta func() any
}

func NewIngressClassV1() resource.Resource {
	return &IngressClassV1{}
}

func (r *IngressClassV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingress_class_v1"
}

func (r *IngressClassV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected func() any, got %T.", req.ProviderData))
		return
	}
	r.SDKv2Meta = meta
}

func (r *IngressClassV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("name"), req, resp)
}

func (r *IngressClassV1) UpgradeIdentity(context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeIdentity("IngressClass", "networking.k8s.io/v1")
}

// Frozen schema-version-0 state of the SDKv2 kubernetes_ingress_class alias.
type ingressClassStateV0 struct {
	ID       string `json:"id"`
	Metadata []struct {
		Annotations     map[string]string `json:"annotations"`
		Labels          map[string]string `json:"labels"`
		GenerateName    string            `json:"generate_name"`
		Generation      int64             `json:"generation"`
		Name            string            `json:"name"`
		ResourceVersion string            `json:"resource_version"`
		UID             string            `json:"uid"`
	} `json:"metadata"`
	Spec []struct {
		Controller string `json:"controller"`
		Parameters []struct {
			APIGroup  string `json:"api_group"`
			Kind      string `json:"kind"`
			Name      string `json:"name"`
			Scope     string `json:"scope"`
			Namespace string `json:"namespace"`
		} `json:"parameters"`
	} `json:"spec"`
}

func (r *IngressClassV1) MoveState(context.Context) []resource.StateMover {
	return []resource.StateMover{{
		StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
			if req.SourceTypeName != "kubernetes_ingress_class" || req.SourceSchemaVersion != 0 ||
				!strings.HasSuffix(req.SourceProviderAddress, "/hashicorp/kubernetes") {
				tflog.Debug(ctx, "Skipping unsupported IngressClass state move", map[string]any{
					"source_type": req.SourceTypeName, "source_version": req.SourceSchemaVersion,
					"source_provider": req.SourceProviderAddress,
				})
				return
			}
			const summary = "Unable to move kubernetes_ingress_class state"
			if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
				resp.Diagnostics.AddError(summary, "The source state must contain JSON data.")
				return
			}
			var source ingressClassStateV0
			if err := json.Unmarshal(req.SourceRawState.JSON, &source); err != nil {
				resp.Diagnostics.AddError(summary, err.Error())
				return
			}
			if len(source.Metadata) != 1 || len(source.Spec) != 1 || source.ID == "" {
				resp.Diagnostics.AddError(summary, "The source state must contain an id and exactly one metadata and spec element.")
				return
			}
			m := source.Metadata[0]
			if m.Name != source.ID {
				resp.Diagnostics.AddError(summary, "The source id does not match metadata.name.")
				return
			}
			metadata := common.MetadataModel{
				MetadataBase: common.MetadataBase{
					Name: types.StringValue(m.Name), Generation: types.Int64Value(m.Generation),
					ResourceVersion: types.StringValue(m.ResourceVersion), UID: types.StringValue(m.UID),
				},
				GenerateName: types.StringValue(m.GenerateName),
			}
			var diags diag.Diagnostics
			metadata.Annotations, diags = types.MapValueFrom(ctx, types.StringType, m.Annotations)
			resp.Diagnostics.Append(diags...)
			metadata.Labels, diags = types.MapValueFrom(ctx, types.StringType, m.Labels)
			resp.Diagnostics.Append(diags...)
			spec := IngressClassV1SpecModel{
				Controller: types.StringValue(source.Spec[0].Controller),
				Parameters: []IngressClassV1ParametersModel{},
			}
			for _, p := range source.Spec[0].Parameters {
				spec.Parameters = append(spec.Parameters, IngressClassV1ParametersModel{
					APIGroup: types.StringValue(p.APIGroup), Kind: types.StringValue(p.Kind),
					Name: types.StringValue(p.Name), Scope: types.StringValue(p.Scope),
					Namespace: types.StringValue(p.Namespace),
				})
			}
			if resp.Diagnostics.HasError() {
				return
			}
			resp.Diagnostics.Append(resp.TargetState.Set(ctx, IngressClassV1Model{
				ID: types.StringValue(source.ID), Metadata: []common.MetadataModel{metadata},
				Spec: []IngressClassV1SpecModel{spec},
			})...)
			if !resp.Diagnostics.HasError() && resp.TargetIdentity != nil {
				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, ingressClassIdentity(m.Name))...)
			}
		},
	}}
}

func ingressClassIdentity(name string) common.ResourceIdentity {
	return common.ResourceIdentity{
		Name: types.StringValue(name), Kind: types.StringValue("IngressClass"),
		APIVersion: types.StringValue("networking.k8s.io/v1"),
	}
}
