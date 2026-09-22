// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
)

var (
	_ resource.Resource                = (*RuntimeClassV1)(nil)
	_ resource.ResourceWithConfigure   = (*RuntimeClassV1)(nil)
	_ resource.ResourceWithImportState = (*RuntimeClassV1)(nil)
	_ resource.ResourceWithIdentity    = (*RuntimeClassV1)(nil)
)

type RuntimeClassV1 struct {
	SDKv2Meta         func() any
	IgnoreAnnotations []string
	IgnoreLabels      []string
}

// kubeIgnoreKeys is a local interface satisfied by kubernetes.providerMetadata.
// It allows the nodev1 package to retrieve the user-configured ignore patterns
// without importing the unexported providerMetadata type.
type kubeIgnoreKeys interface {
	IgnoreAnnotationPatterns() []string
	IgnoreLabelPatterns() []string
}

func NewRuntimeClassV1() resource.Resource {
	return &RuntimeClassV1{}
}

func (r *RuntimeClassV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runtime_class_v1"
}

func (r *RuntimeClassV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.SDKv2Meta = req.ProviderData.(func() any)
	if ik, ok := r.SDKv2Meta().(kubeIgnoreKeys); ok {
		r.IgnoreAnnotations = ik.IgnoreAnnotationPatterns()
		r.IgnoreLabels = ik.IgnoreLabelPatterns()
	}
}

func (r *RuntimeClassV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"api_version": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"kind": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"name": identityschema.StringAttribute{
				RequiredForImport: true,
			},
		},
	}
}
