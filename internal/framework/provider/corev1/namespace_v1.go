// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"k8s.io/client-go/kubernetes"

	provkubernetes "github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

var (
	_ resource.Resource                 = (*NamespaceV1)(nil)
	_ resource.ResourceWithConfigure    = (*NamespaceV1)(nil)
	_ resource.ResourceWithIdentity     = (*NamespaceV1)(nil)
	_ resource.ResourceWithImportState  = (*NamespaceV1)(nil)
	_ resource.ResourceWithUpgradeState = (*NamespaceV1)(nil)
	_ resource.ResourceWithMoveState    = (*NamespaceV1)(nil)
)

// NamespaceV1 is the Plugin Framework implementation of kubernetes_namespace_v1.
//
// The deprecated alias kubernetes_namespace deliberately stays on the SDKv2
// server with its existing implementation; a resource type may be served by
// exactly one mux server, and only the _v1 name moves here. Both data sources
// also stay on SDKv2 — data sources and managed resources occupy separate
// namespaces, so this raises no duplicate-type conflict.
type NamespaceV1 struct {
	// SDKv2Meta yields the configured SDKv2 provider meta, which is where the
	// Kubernetes clients and the provider-level ignore_annotations /
	// ignore_labels settings live (rule K8S-CRUD-001). A Framework resource in
	// this provider never builds its own client: doing so would silently ignore
	// every provider-level auth setting the practitioner configured.
	SDKv2Meta func() any
}

func NewNamespaceV1() resource.Resource { return &NamespaceV1{} }

func (r *NamespaceV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_namespace_v1"
}

func (r *NamespaceV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		// Normal during early provider graph walks; CRUD re-checks for a usable
		// client rather than treating this as an error.
		return
	}
	meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider configuration",
			"Expected the Kubernetes metadata callback. This is a provider bug.",
		)
		return
	}
	r.SDKv2Meta = meta
}

// IdentitySchema reproduces resourceIdentitySchemaNonNamespaced() exactly,
// including its Version (rule K8S-MIGRATE-021).
//
// Version is explicit: identity data written by the SDKv2 implementation is
// read back by this one, and the Framework would otherwise default to 0 against
// stored identity version 1. The admissionregistrationv1 reference omits
// Version because it is greenfield; copying it here would break every existing
// import. This also overrides K8S-SCHEMA-004's "all RequiredForImport" default —
// for a migration the SDKv2 identity is the contract, and here all three
// attributes happen to be RequiredForImport anyway.
func (r *NamespaceV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Version: 1,
		Attributes: map[string]identityschema.Attribute{
			"name":        identityschema.StringAttribute{RequiredForImport: true},
			"api_version": identityschema.StringAttribute{RequiredForImport: true},
			"kind":        identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}

// client returns the Kubernetes clientset plus the provider's metadata-ignore
// policy, or reports a diagnostic if the provider was not configured.
func (r *NamespaceV1) client(diags interface{ AddError(string, string) }) (*kubernetes.Clientset, []string, []string, bool) {
	if r.SDKv2Meta == nil {
		diags.AddError("Provider not configured",
			"The Kubernetes client is unavailable. This is a provider bug.")
		return nil, nil, nil, false
	}
	meta := r.SDKv2Meta()
	clients, ok := meta.(provkubernetes.KubeClientsets)
	if !ok {
		diags.AddError("Provider not configured",
			"The SDKv2 provider meta did not implement KubeClientsets. This is a provider bug.")
		return nil, nil, nil, false
	}
	conn, err := clients.MainClientset()
	if err != nil {
		diags.AddError("kubernetes client error", err.Error())
		return nil, nil, nil, false
	}
	ignoreAnnotations, ignoreLabels := provkubernetes.IgnoredMetadataKeys(meta)
	return conn, ignoreAnnotations, ignoreLabels, true
}
