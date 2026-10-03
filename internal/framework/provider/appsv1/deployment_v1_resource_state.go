// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// UpgradeState accepts the SDKv2 schema versions 0 and 1.
func (d *DeploymentV1) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	upgrader := func(version int64) resource.StateUpgrader {
		return resource.StateUpgrader{
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				state, diagnostics := d.decodeHistoricalState(ctx, req.RawState, version)
				resp.Diagnostics.Append(diagnostics...)
				if !resp.Diagnostics.HasError() {
					resp.State = state
				}
			},
		}
	}
	return map[int64]resource.StateUpgrader{0: upgrader(0), 1: upgrader(1)}
}

func (d *DeploymentV1) MoveState(context.Context) []resource.StateMover {
	return []resource.StateMover{{
		StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
			if !isSDKv2SourceType(req, "kubernetes_deployment") {
				tflog.Debug(ctx, "Skipping unsupported Deployment move source", map[string]any{
					"type": req.SourceTypeName, "version": req.SourceSchemaVersion, "provider": req.SourceProviderAddress,
				})
				return
			}
			state, diagnostics := d.decodeHistoricalState(ctx, req.SourceRawState, req.SourceSchemaVersion)
			resp.Diagnostics.Append(diagnostics...)
			if resp.Diagnostics.HasError() {
				return
			}
			var model DeploymentV1Model
			resp.Diagnostics.Append(state.Get(ctx, &model)...)
			if resp.Diagnostics.HasError() {
				return
			}
			namespace, name, err := kubernetes.IdParts(model.ID.ValueString())
			if err != nil || namespace == "" || name == "" {
				resp.Diagnostics.AddError(moveStateErrSummary, "Source state must contain a namespace/name ID.")
				return
			}
			resp.TargetState = state
			if resp.TargetIdentity != nil {
				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, common.NamespacedResourceIdentity{
					ResourceIdentity: common.ResourceIdentity{
						APIVersion: types.StringValue(deploymentAPIVersion),
						Kind:       types.StringValue(deploymentKind),
						Name:       types.StringValue(name),
					},
					Namespace: types.StringValue(namespace),
				})...)
			}
		},
	}}
}

func (d *DeploymentV1) decodeHistoricalState(ctx context.Context, raw *tfprotov6.RawState, version int64) (tfsdk.State, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	var response resource.SchemaResponse
	d.Schema(ctx, resource.SchemaRequest{}, &response)
	diagnostics.Append(response.Diagnostics...)
	state := tfsdk.State{Schema: response.Schema}
	var rewrite func(map[string]any) error
	if version == 0 {
		rewrite = func(values map[string]any) error {
			return upgradeDeploymentV0Resources(values, "state", []string{"spec", "template", "spec"})
		}
	}
	value, err := common.DecodeLegacyState(ctx, raw, response.Schema, rewrite)
	if err != nil {
		diagnostics.AddError("Unable to decode legacy deployment state", err.Error())
		return state, diagnostics
	}
	state.Raw = value
	return state, diagnostics
}

func upgradeDeploymentV0Resources(object map[string]any, location string, path []string) error {
	if object == nil {
		return fmt.Errorf("%s must be an object", location)
	}
	if len(path) > 0 {
		name := path[0]
		value, exists := object[name]
		if !exists || value == nil {
			return nil
		}
		list, ok := value.([]any)
		if !ok || len(list) > 1 {
			return fmt.Errorf("%s.%s must be a list with at most one object", location, name)
		}
		for _, entry := range list {
			child, ok := entry.(map[string]any)
			if !ok {
				return fmt.Errorf("%s.%s[0] must be an object", location, name)
			}
			if err := upgradeDeploymentV0Resources(child, location+"."+name+"[0]", path[1:]); err != nil {
				return err
			}
		}
		return nil
	}
	for _, name := range []string{"container", "init_container"} {
		value := object[name]
		if value == nil {
			continue
		}
		containers, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%s.%s must be a list", location, name)
		}
		for i, entry := range containers {
			container, ok := entry.(map[string]any)
			if !ok {
				return fmt.Errorf("%s.%s[%d] must be an object", location, name, i)
			}
			if container["resources"] == nil {
				continue
			}
			resources, ok := container["resources"].([]any)
			if !ok || len(resources) > 1 {
				return fmt.Errorf("%s.%s[%d].resources must contain at most one object", location, name, i)
			}
			for _, entry := range resources {
				resourceMap, ok := entry.(map[string]any)
				if !ok {
					return fmt.Errorf("%s.%s[%d].resources[0] must be an object", location, name, i)
				}
				for _, field := range []string{"limits", "requests"} {
					value := resourceMap[field]
					if value == nil {
						resourceMap[field] = map[string]any{}
						continue
					}
					list, ok := value.([]any)
					if !ok || len(list) > 1 {
						return fmt.Errorf("%s.%s[%d].resources[0].%s must be a legacy singleton list", location, name, i, field)
					}
					converted := map[string]any{}
					if len(list) == 1 {
						converted, ok = list[0].(map[string]any)
						if !ok {
							return fmt.Errorf("%s.%s[%d].resources[0].%s[0] must be an object", location, name, i, field)
						}
					}
					resourceMap[field] = converted
				}
			}
		}
	}
	return nil
}
