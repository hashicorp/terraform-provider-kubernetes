// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

var (
	_ resource.Resource                = (*NamespaceV1)(nil)
	_ resource.ResourceWithConfigure   = (*NamespaceV1)(nil)
	_ resource.ResourceWithIdentity    = (*NamespaceV1)(nil)
	_ resource.ResourceWithImportState = (*NamespaceV1)(nil)
	_ resource.ResourceWithMoveState   = (*NamespaceV1)(nil)
)

type NamespaceV1 struct {
	// SDKv2Meta must stay func() any: that is the concrete type stored in
	// ProviderData by internal/framework/provider/provider_configure.go. Go
	// function types are invariant, so asserting to func() kubernetes.KubeClientsets
	// compiles but panics at runtime. Assert the *result* instead — see meta().
	SDKv2Meta func() any
}

func NewNamespaceV1() resource.Resource {
	return &NamespaceV1{}
}

// Metadata implements [resource.Resource].
func (n *NamespaceV1) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_namespace_v1"
}

// Configure implements [resource.ResourceWithConfigure].
func (n *NamespaceV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	n.SDKv2Meta = sdkv2Meta
}

// meta resolves the SDKv2 provider metadata as API clients. The call is deferred
// until now rather than made in Configure because the SDKv2 provider is configured
// independently by the mux server, and its meta is not populated until that happens.
func (n *NamespaceV1) meta() kubernetes.KubeClientsets {
	return n.SDKv2Meta().(kubernetes.KubeClientsets)
}

// ImportState implements [resource.ResourceWithImportState].
func (n *NamespaceV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("name"), req, resp)
}

// sdkv2ProviderAddressSuffix matches the source provider in a MoveState request. The
// address is HOSTNAME/NAMESPACE/TYPE; the hostname is deliberately ignored, since the same
// provider is registry.terraform.io for most users and a mirror or private registry for
// others.
const sdkv2ProviderAddressSuffix = "hashicorp/kubernetes"

// MoveState implements [resource.ResourceWithMoveState], supporting
//
//	moved {
//	  from = kubernetes_namespace.example
//	  to   = kubernetes_namespace_v1.example
//	}
//
// which is the only supported route off the deprecated unversioned alias: it stays on SDKv2
// while this resource is served by the framework, and Terraform can move state across
// resource types only if the *destination* provider implements this.
//
// No field-by-field translation is needed today — the SDKv2 shape and this resource's schema
// coincide — so the decoded source model is written straight back out. If the two ever
// diverge, this is where the translation goes.
func (n *NamespaceV1) MoveState(ctx context.Context) []resource.StateMover {

	schemaResp := &resource.SchemaResponse{}
	n.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	return []resource.StateMover{
		{
			SourceSchema: &schemaResp.Schema,
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				// Return without state or diagnostics when this is not our move: that is
				// how the framework marks a mover as skipped so it can try the next one.
				// An error here would break unrelated moves.
				if req.SourceTypeName != "kubernetes_namespace" ||
					req.SourceSchemaVersion != 0 ||
					!strings.HasSuffix(req.SourceProviderAddress, sdkv2ProviderAddressSuffix) {
					return
				}

				// Past this point the request is ours, so problems are real errors.
				if req.SourceState == nil {
					resp.Diagnostics.AddError(
						"Unable to move kubernetes_namespace state",
						"The source state was not populated from SourceSchema. This is a bug in the provider.",
					)
					return
				}

				var state NamespaceV1Model
				resp.Diagnostics.Append(req.SourceState.Get(ctx, &state)...)
				if resp.Diagnostics.HasError() {
					return
				}

				resp.Diagnostics.Append(resp.TargetState.Set(ctx, &state)...)
				if resp.Diagnostics.HasError() {
					return
				}

				// req.SourceIdentity is a raw *tfprotov6.RawState rather than a decoded
				// identity, and decoding it would buy nothing: every field is derivable
				// here. api_version and kind are constants for this resource, and the name
				// is in the state just moved — so build it exactly as Create and Read do.
				// This also covers state written before resource identity existed.
				if resp.TargetIdentity == nil {
					return
				}

				name := state.ID
				if len(state.Metadata) > 0 && !state.Metadata[0].Name.IsNull() {
					name = state.Metadata[0].Name
				}

				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, NamespaceResourceIdentity{
					APIVersion: types.StringValue(namespaceAPIVersion),
					Kind:       types.StringValue(namespaceKind),
					Name:       name,
				})...)
			},
		},
	}
}
