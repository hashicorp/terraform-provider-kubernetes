// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

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
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	_ resource.Resource                    = (*EndpointsV1)(nil)
	_ resource.ResourceWithConfigure       = (*EndpointsV1)(nil)
	_ resource.ResourceWithIdentity        = (*EndpointsV1)(nil)
	_ resource.ResourceWithImportState     = (*EndpointsV1)(nil)
	_ resource.ResourceWithMoveState       = (*EndpointsV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*EndpointsV1)(nil)
)

const (
	endpointsAPIVersion          = "v1"
	endpointsKind                = "Endpoints"
	moveEndpointsStateErrSummary = "Unable to move kubernetes_endpoints state"
)

type EndpointsV1 struct {
	SDKv2Meta func() any
}

func NewEndpointsV1() resource.Resource {
	return &EndpointsV1{}
}

// Metadata implements [resource.Resource].
func (e *EndpointsV1) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_endpoints_v1"
}

// Configure implements [resource.ResourceWithConfigure].
func (e *EndpointsV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	sdkv2Meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data",
			fmt.Sprintf("Expected func() any, got %T. This is a bug in the provider.", req.ProviderData),
		)
		return
	}
	e.SDKv2Meta = sdkv2Meta
}

func (e *EndpointsV1) sdkv2Meta() (kubernetes.KubeClientsets, kubernetes.MetadataFilters, diag.Diagnostics) {
	return common.ResolveSDKv2Meta(e.SDKv2Meta)
}

// ImportState implements [resource.ResourceWithImportState].
func (e *EndpointsV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	if id == "" {
		var identityData common.NamespacedResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identityData.Kind.ValueString() != endpointsKind || identityData.APIVersion.ValueString() != endpointsAPIVersion {
			resp.Diagnostics.AddError("invalid identity import",
				fmt.Sprintf("Expected kind %q and api_version %q, got kind %q and api_version %q.",
					endpointsKind, endpointsAPIVersion, identityData.Kind.ValueString(), identityData.APIVersion.ValueString()))
			return
		}
		namespace := identityData.Namespace.ValueString()
		if namespace == "" {
			namespace = metav1.NamespaceDefault
		}
		id = kubernetes.BuildId(metav1.ObjectMeta{Namespace: namespace, Name: identityData.Name.ValueString()})
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// MoveState implements [resource.ResourceWithMoveState].
func (e *EndpointsV1) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		{
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				if req.SourceTypeName != "kubernetes_endpoints" ||
					req.SourceSchemaVersion != 0 ||
					!strings.HasSuffix(req.SourceProviderAddress, sdkv2ProviderAddressSuffix) {
					tflog.Debug(ctx, "MoveState: not a kubernetes_endpoints v0 source, skipping", map[string]any{
						"source_type_name":        req.SourceTypeName,
						"source_schema_version":   req.SourceSchemaVersion,
						"source_provider_address": req.SourceProviderAddress,
					})
					return
				}

				if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
					resp.Diagnostics.AddError(moveEndpointsStateErrSummary,
						"The source state has no JSON data. Flatmap state predates Terraform 0.12 and is not supported.")
					return
				}

				var prior sdkv2EndpointsStateV0
				if err := json.Unmarshal(req.SourceRawState.JSON, &prior); err != nil {
					resp.Diagnostics.AddError(moveEndpointsStateErrSummary, fmt.Sprintf("Could not decode the source state: %s", err))
					return
				}
				if prior.ID == "" {
					resp.Diagnostics.AddError(moveEndpointsStateErrSummary, "The source state has an empty id.")
					return
				}
				if len(prior.Metadata) != 1 {
					resp.Diagnostics.AddError(moveEndpointsStateErrSummary,
						fmt.Sprintf("Expected exactly 1 metadata element in the source state, got %d.", len(prior.Metadata)))
					return
				}
				m := prior.Metadata[0]

				var diags diag.Diagnostics
				meta := common.NamespacedMetadataModel{
					MetadataModel: common.MetadataModel{
						MetadataBase: common.MetadataBase{
							Name:            types.StringValue(m.Name),
							Generation:      types.Int64Value(m.Generation),
							ResourceVersion: types.StringValue(m.ResourceVersion),
							UID:             types.StringValue(m.UID),
						},
						GenerateName: types.StringNull(),
					},
					Namespace: types.StringValue(m.Namespace),
				}
				meta.Labels, diags = common.SDKv2MapToFramework(ctx, m.Labels)
				resp.Diagnostics.Append(diags...)
				meta.Annotations, diags = common.SDKv2MapToFramework(ctx, m.Annotations)
				resp.Diagnostics.Append(diags...)

				if m.GenerateName != "" {
					meta.GenerateName = types.StringValue(m.GenerateName)
				}

				var subsets []EndpointsSubsetModel
				if len(prior.Subset) > 0 {
					subsets = make([]EndpointsSubsetModel, 0, len(prior.Subset))
					for _, s := range prior.Subset {
						var addresses []EndpointsAddressModel
						for _, a := range s.Address {
							addr := EndpointsAddressModel{
								IP:       types.StringValue(a.IP),
								Hostname: types.StringNull(),
								NodeName: types.StringNull(),
							}
							if a.Hostname != "" {
								addr.Hostname = types.StringValue(a.Hostname)
							}
							if a.NodeName != "" {
								addr.NodeName = types.StringValue(a.NodeName)
							}
							addresses = append(addresses, addr)
						}

						var notReadyAddresses []EndpointsAddressModel
						for _, a := range s.NotReadyAddress {
							addr := EndpointsAddressModel{
								IP:       types.StringValue(a.IP),
								Hostname: types.StringNull(),
								NodeName: types.StringNull(),
							}
							if a.Hostname != "" {
								addr.Hostname = types.StringValue(a.Hostname)
							}
							if a.NodeName != "" {
								addr.NodeName = types.StringValue(a.NodeName)
							}
							notReadyAddresses = append(notReadyAddresses, addr)
						}

						var ports []EndpointsPortModel
						for _, p := range s.Port {
							port := EndpointsPortModel{
								Port:     types.Int64Value(p.Port),
								Name:     types.StringNull(),
								Protocol: types.StringValue("TCP"),
							}
							if p.Name != "" {
								port.Name = types.StringValue(p.Name)
							}
							if p.Protocol != "" {
								port.Protocol = types.StringValue(p.Protocol)
							}
							ports = append(ports, port)
						}

						subsets = append(subsets, EndpointsSubsetModel{
							Address:         addresses,
							NotReadyAddress: notReadyAddresses,
							Port:            ports,
						})
					}
				}

				state := EndpointsV1Model{
					ID:       types.StringValue(prior.ID),
					Metadata: []common.NamespacedMetadataModel{meta},
					Subset:   subsets,
				}

				resp.Diagnostics.Append(resp.TargetState.Set(ctx, &state)...)
				if resp.Diagnostics.HasError() || resp.TargetIdentity == nil {
					return
				}

				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, common.NamespacedResourceIdentity{
					ResourceIdentity: common.ResourceIdentity{
						APIVersion: types.StringValue(endpointsAPIVersion),
						Kind:       types.StringValue(endpointsKind),
						Name:       types.StringValue(m.Name),
					},
					Namespace: types.StringValue(m.Namespace),
				})...)
			},
		},
	}
}

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity].
func (e *EndpointsV1) UpgradeIdentity(ctx context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(endpointsKind, endpointsAPIVersion)
}
