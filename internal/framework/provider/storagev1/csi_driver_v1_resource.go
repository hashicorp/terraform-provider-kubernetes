// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1

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
	_ resource.Resource                    = (*CSIDriverV1)(nil)
	_ resource.ResourceWithConfigure       = (*CSIDriverV1)(nil)
	_ resource.ResourceWithImportState     = (*CSIDriverV1)(nil)
	_ resource.ResourceWithIdentity        = (*CSIDriverV1)(nil)
	_ resource.ResourceWithMoveState       = (*CSIDriverV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*CSIDriverV1)(nil)
)

// client-go clears TypeMeta on typed responses, so identity cannot read these off the object.
const (
	csiDriverAPIVersion = "storage.k8s.io/v1"
	csiDriverKind       = "CSIDriver"
)

type CSIDriverV1 struct {
	SDKv2Meta func() any
}

func NewCSIDriverV1() resource.Resource {
	return &CSIDriverV1{}
}

// Metadata implements [resource.Resource].
func (r *CSIDriverV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_csi_driver_v1"
}

// Configure implements [resource.ResourceWithConfigure].
func (r *CSIDriverV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	sdkv2Meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected provider data to be of type 'func() any', got: %T", req.ProviderData),
		)
		return
	}
	r.SDKv2Meta = sdkv2Meta
}

// ImportState implements [resource.ResourceWithImportState].
func (r *CSIDriverV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("name"), req, resp)
}

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity].
//
// The SDKv2 resource never declared an identity, so all existing state, from any release,
// carries identity version 0 and no identity. Terraform asks for an upgrade whenever the
// stored version differs from the declared one, and the framework fails the plan with
// "Unable to Upgrade Resource Identity" unless the resource supplies an upgrader.
func (r *CSIDriverV1) UpgradeIdentity(_ context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeIdentity(csiDriverKind, csiDriverAPIVersion)
}

func csiDriverIdentity(name string) common.ResourceIdentity {
	return common.ResourceIdentity{
		Name:       types.StringValue(name),
		Kind:       types.StringValue(csiDriverKind),
		APIVersion: types.StringValue(csiDriverAPIVersion),
	}
}

const (
	deprecatedCSIDriverTypeName = "kubernetes_csi_driver"
	moveStateErrSummary         = "Unable to move kubernetes_csi_driver state"

	// The source provider address is HOSTNAME/NAMESPACE/TYPE. The hostname differs for
	// mirrors and private registries, so it is not compared; the leading slash keeps a
	// fork such as nothashicorp/kubernetes from matching.
	sdkv2ProviderAddressSuffix = "/hashicorp/kubernetes"
)

// sdkv2CSIDriverStateV0 is kubernetes_csi_driver state as SDKv2 stores it at schema
// version 0. It describes stored JSON rather than this resource's schema, so it must not
// follow schema changes. SDKv2 writes "" and false for unset primitives; maps stay Go
// maps because SDKv2 does distinguish a null map from an empty one.
type sdkv2CSIDriverStateV0 struct {
	ID       string                     `json:"id"`
	Metadata []sdkv2CSIDriverMetadataV0 `json:"metadata"`
	Spec     []sdkv2CSIDriverSpecV0     `json:"spec"`
}

type sdkv2CSIDriverMetadataV0 struct {
	Annotations     map[string]string `json:"annotations"`
	GenerateName    string            `json:"generate_name"`
	Generation      int64             `json:"generation"`
	Labels          map[string]string `json:"labels"`
	Name            string            `json:"name"`
	ResourceVersion string            `json:"resource_version"`
	UID             string            `json:"uid"`
}

type sdkv2CSIDriverSpecV0 struct {
	AttachRequired       *bool    `json:"attach_required"`
	PodInfoOnMount       bool     `json:"pod_info_on_mount"`
	VolumeLifecycleModes []string `json:"volume_lifecycle_modes"`
}

// MoveState implements [resource.ResourceWithMoveState] for
//
//	moved {
//	  from = kubernetes_csi_driver.example
//	  to   = kubernetes_csi_driver_v1.example
//	}
//
// the route off the deprecated alias, which stays on SDKv2. Terraform 1.8 or later is
// required for a move between resource types.
func (r *CSIDriverV1) MoveState(_ context.Context) []resource.StateMover {
	// No SourceSchema: the framework would decode the source against it before calling
	// the mover, so the frozen struct above is decoded by hand instead.
	return []resource.StateMover{{StateMover: moveFromDeprecatedCSIDriver}}
}

func moveFromDeprecatedCSIDriver(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	// SourceIdentitySchemaVersion is not checked: terraform-plugin-go never copies it off
	// the wire, so it is always 0.
	if req.SourceTypeName != deprecatedCSIDriverTypeName ||
		req.SourceSchemaVersion != 0 ||
		!strings.HasSuffix(req.SourceProviderAddress, sdkv2ProviderAddressSuffix) {
		// Returning nothing skips this mover, and the framework then fails the move
		// without saying why, so leave the reason in the log.
		tflog.Debug(ctx, "MoveState: not a kubernetes_csi_driver v0 source, skipping", map[string]any{
			"source_type_name":        req.SourceTypeName,
			"source_schema_version":   req.SourceSchemaVersion,
			"source_provider_address": req.SourceProviderAddress,
		})
		return
	}

	if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
		resp.Diagnostics.AddError(moveStateErrSummary, "The source state has no JSON data. Flatmap state is not supported.")
		return
	}
	var source sdkv2CSIDriverStateV0
	if err := json.Unmarshal(req.SourceRawState.JSON, &source); err != nil {
		resp.Diagnostics.AddError(moveStateErrSummary, fmt.Sprintf("Could not decode the source state: %s", err))
		return
	}
	if source.ID == "" || len(source.Metadata) != 1 {
		resp.Diagnostics.AddError(moveStateErrSummary, fmt.Sprintf(
			"Expected a non-empty id and exactly 1 metadata element in the source state, got id %q and %d elements.",
			source.ID, len(source.Metadata)))
		return
	}

	state, diags := csiDriverStateFromSDKv2(ctx, source)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() || resp.TargetIdentity == nil {
		return
	}
	// Built rather than taken from req.SourceIdentity: the alias never had an identity.
	resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, csiDriverIdentity(source.Metadata[0].Name))...)
}

func csiDriverStateFromSDKv2(ctx context.Context, source sdkv2CSIDriverStateV0) (CSIDriverV1Model, diag.Diagnostics) {
	sourceMetadata := source.Metadata[0]
	metadata := common.MetadataModel{
		MetadataBase: common.MetadataBase{
			Name:            types.StringValue(sourceMetadata.Name),
			Generation:      types.Int64Value(sourceMetadata.Generation),
			ResourceVersion: types.StringValue(sourceMetadata.ResourceVersion),
			UID:             types.StringValue(sourceMetadata.UID),
		},
		// SDKv2 stores an unset generate_name as "". Left as "", a plan with
		// -refresh=false compares it against null and replaces the driver.
		GenerateName: types.StringNull(),
	}
	if sourceMetadata.GenerateName != "" {
		metadata.GenerateName = types.StringValue(sourceMetadata.GenerateName)
	}

	var diags, d diag.Diagnostics
	metadata.Labels, d = common.SDKv2MapToFramework(ctx, sourceMetadata.Labels)
	diags.Append(d...)
	metadata.Annotations, d = common.SDKv2MapToFramework(ctx, sourceMetadata.Annotations)
	diags.Append(d...)

	spec := make([]CSIDriverV1SpecModel, 0, len(source.Spec))
	for _, sourceSpec := range source.Spec {
		modes, d := types.ListValueFrom(ctx, types.StringType, sourceSpec.VolumeLifecycleModes)
		diags.Append(d...)
		spec = append(spec, CSIDriverV1SpecModel{
			AttachRequired:       types.BoolPointerValue(sourceSpec.AttachRequired),
			PodInfoOnMount:       types.BoolValue(sourceSpec.PodInfoOnMount),
			VolumeLifecycleModes: modes,
		})
	}

	return CSIDriverV1Model{
		ID:       types.StringValue(source.ID),
		Metadata: []common.MetadataModel{metadata},
		Spec:     spec,
	}, diags
}
